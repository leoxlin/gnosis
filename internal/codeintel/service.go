package codeintel

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"sync"

	"gnosis/internal/vault"
)

// Service owns current code reads and any live workspaces for one host.
type Service struct {
	workspace string
	live      map[string]*Workspace
	closeOnce sync.Once
	closeErr  error
}

// NewService creates a current-read module without starting live indexing.
func NewService(workspace string) *Service {
	return &Service{workspace: workspace, live: map[string]*Workspace{}}
}

func OpenService(ctx context.Context, workspace string) (*Service, error) {
	configured, err := vault.CodeScopes(workspace)
	if err != nil {
		return nil, err
	}
	service := NewService(workspace)
	for _, scope := range configured {
		if !scope.Live {
			continue
		}
		opened, err := OpenWorkspace(ctx, LiveConfig{Workspace: workspace, Scope: scope.Name})
		if err != nil {
			return nil, errors.Join(err, service.Close())
		}
		service.live[scope.Name] = opened
	}
	return service, nil
}

func (service *Service) Status(ctx context.Context, scope string) (StatusResult, error) {
	if strings.TrimSpace(scope) == "" {
		return StatusResult{}, errors.New("scope is required")
	}
	if service.live == nil {
		return StatusResult{}, os.ErrClosed
	}
	if workspace := service.live[scope]; workspace != nil {
		return workspace.Status(), nil
	}
	reader, err := openReader(service.workspace, scope)
	if err != nil {
		return StatusResult{}, err
	}
	status := reader.Status()
	if err := reader.checkCurrent(ctx); err != nil {
		status.Status = "not_current"
	}
	return status, reader.close()
}

func (service *Service) Search(ctx context.Context, scope, query, language string, limit int) (result SearchResult, err error) {
	err = service.readCurrent(ctx, scope, func(reader *Reader) error {
		result = reader.Search(query, language, limit)
		return nil
	})
	return result, err
}

func (service *Service) ReadSymbol(ctx context.Context, scope, id string) (result SymbolResult, err error) {
	err = service.readCurrent(ctx, scope, func(reader *Reader) error {
		result, err = reader.ReadSymbol(id)
		return err
	})
	return result, err
}

func (service *Service) Diagnostics(ctx context.Context, scope, path, language, category string, limit int) (result DiagnosticResult, err error) {
	err = service.readCurrent(ctx, scope, func(reader *Reader) error {
		result = reader.Diagnostics(path, language, category, limit)
		return nil
	})
	return result, err
}

func (service *Service) Trace(ctx context.Context, scope, id, direction string, limit int) (result TraceResult, err error) {
	err = service.readCurrent(ctx, scope, func(reader *Reader) error {
		result, err = reader.Trace(id, direction, limit)
		return err
	})
	return result, err
}

func (service *Service) Neighbors(ctx context.Context, scope, id, direction string, limit int) (result TraceResult, err error) {
	err = service.readCurrent(ctx, scope, func(reader *Reader) error {
		result, err = reader.Neighbors(id, direction, limit)
		return err
	})
	return result, err
}

func (service *Service) Path(ctx context.Context, scope, from, to, direction string, depth, limit int) (result TraceResult, err error) {
	err = service.readCurrent(ctx, scope, func(reader *Reader) error {
		result, err = reader.Path(from, to, direction, depth, limit)
		return err
	})
	return result, err
}

func (service *Service) readCurrent(ctx context.Context, scope string, callback func(*Reader) error) error {
	if strings.TrimSpace(scope) == "" {
		return errors.New("scope is required")
	}
	if callback == nil {
		return errors.New("read callback is required")
	}
	if service.live == nil {
		return os.ErrClosed
	}
	if workspace := service.live[scope]; workspace != nil {
		return workspace.ReadCurrent(ctx, callback)
	}
	reader, err := openReader(service.workspace, scope)
	if err != nil {
		return err
	}
	if err := reader.checkCurrent(ctx); err != nil {
		return errors.Join(err, reader.close())
	}
	err = callback(reader)
	closeErr := reader.close()
	return errors.Join(err, closeErr)
}

func (service *Service) Close() error {
	service.closeOnce.Do(func() {
		failures := make([]error, 0, len(service.live))
		scopes := make([]string, 0, len(service.live))
		for scope := range service.live {
			scopes = append(scopes, scope)
		}
		slices.Sort(scopes)
		for _, scope := range scopes {
			workspace := service.live[scope]
			if err := workspace.Close(); err != nil {
				failures = append(failures, fmt.Errorf("close code scope %q: %w", scope, err))
			}
		}
		service.live = nil
		service.closeErr = errors.Join(failures...)
	})
	return service.closeErr
}
