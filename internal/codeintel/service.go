package codeintel

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"gnosis/internal/vault"
)

// Service owns the live code workspaces for one protocol host.
type Service struct {
	workspace string
	live      map[string]*Workspace
	closeOnce sync.Once
	closeErr  error
}

func OpenService(ctx context.Context, workspace string) (*Service, error) {
	configured, err := vault.CodeScopes(workspace)
	if err != nil {
		return nil, err
	}
	service := &Service{workspace: workspace, live: map[string]*Workspace{}}
	for _, scope := range configured {
		if !scope.Live {
			continue
		}
		opened, err := OpenWorkspace(ctx, LiveConfig{Workspace: workspace, Scope: scope.Name})
		if err != nil {
			service.Close()
			return nil, err
		}
		service.live[scope.Name] = opened
	}
	return service, nil
}

func (service *Service) Status(ctx context.Context, scope string) (StatusResult, error) {
	if strings.TrimSpace(scope) == "" {
		return StatusResult{}, errors.New("scope is required")
	}
	if workspace := service.live[scope]; workspace != nil {
		return workspace.Status(), nil
	}
	reader, err := Open(service.workspace, scope)
	if err != nil {
		return StatusResult{}, err
	}
	defer reader.Close()
	status := reader.Status()
	if err := reader.CheckCurrent(ctx); err != nil {
		status.Status = "not_current"
	}
	return status, nil
}

func (service *Service) ReadCurrent(ctx context.Context, scope string, callback func(ReadView) error) error {
	if strings.TrimSpace(scope) == "" {
		return errors.New("scope is required")
	}
	if workspace := service.live[scope]; workspace != nil {
		return workspace.ReadCurrent(ctx, callback)
	}
	reader, err := Open(service.workspace, scope)
	if err != nil {
		return err
	}
	if err := reader.CheckCurrent(ctx); err != nil {
		reader.Close()
		return err
	}
	token := &readToken{}
	token.valid.Store(true)
	err = callback(ReadView{reader: reader, token: token})
	token.valid.Store(false)
	closeErr := reader.Close()
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
