package codeintel

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
)

type fsNotifySource struct {
	root       string
	gitRoots   []string
	watcher    *fsnotify.Watcher
	events     chan eventHint
	errors     chan error
	watched    map[string]bool
	maxWatches int
	mu         sync.Mutex
	closeOnce  sync.Once
}

func newFSNotifySource(root string, maxWatches int) (*fsNotifySource, error) {
	canonical, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	canonical, err = filepath.Abs(canonical)
	if err != nil {
		return nil, err
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	source := &fsNotifySource{
		root: filepath.Clean(canonical), watcher: watcher, events: make(chan eventHint, liveEventLimit),
		errors: make(chan error, 1), watched: map[string]bool{}, maxWatches: maxWatches,
	}
	gitRoots, err := resolveGitControlPaths(source.root)
	if err != nil {
		watcher.Close()
		return nil, err
	}
	source.gitRoots = gitRoots
	base := append([]string{source.root, filepath.Dir(source.root)}, gitRoots...)
	for _, path := range base {
		if err := source.addDirectory(path); err != nil {
			watcher.Close()
			return nil, err
		}
	}
	for _, gitRoot := range gitRoots {
		for _, relative := range []string{"refs", filepath.Join("refs", "heads"), filepath.Join("refs", "tags")} {
			path := filepath.Join(gitRoot, relative)
			walkErr := filepath.WalkDir(path, func(current string, entry os.DirEntry, walkErr error) error {
				if walkErr != nil || !entry.IsDir() {
					return walkErr
				}
				return source.addDirectory(current)
			})
			if walkErr != nil && !errors.Is(walkErr, os.ErrNotExist) {
				watcher.Close()
				return nil, walkErr
			}
		}
	}
	go source.run()
	return source, nil
}

func (source *fsNotifySource) Events() <-chan eventHint { return source.events }
func (source *fsNotifySource) Errors() <-chan error     { return source.errors }

func (source *fsNotifySource) WatchDocuments(documents []SourceDocument) error {
	for _, document := range documents {
		path := filepath.Join(source.root, filepath.FromSlash(document.Path))
		parent := filepath.Dir(path)
		if !within(source.root, parent) {
			return fmt.Errorf("tracked source parent escapes repository: %s", document.Path)
		}
		if err := source.addDirectory(parent); err != nil {
			return err
		}
	}
	return nil
}

func (source *fsNotifySource) addDirectory(path string) error {
	path = filepath.Clean(path)
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		path = filepath.Dir(path)
	}
	source.mu.Lock()
	defer source.mu.Unlock()
	if source.watched[path] {
		return nil
	}
	if len(source.watched) >= source.maxWatches {
		return fmt.Errorf("live watched-directory bound exceeded (%d)", source.maxWatches)
	}
	if err := source.watcher.Add(path); err != nil {
		return err
	}
	source.watched[path] = true
	return nil
}

func (source *fsNotifySource) run() {
	defer close(source.events)
	defer close(source.errors)
	for {
		select {
		case event, ok := <-source.watcher.Events:
			if !ok {
				return
			}
			source.emit(source.normalize(event))
		case err, ok := <-source.watcher.Errors:
			if !ok {
				return
			}
			select {
			case source.errors <- fmt.Errorf("filesystem observation: %w", err):
			default:
			}
			source.emit(eventHint{full: true})
		}
	}
}

func (source *fsNotifySource) normalize(event fsnotify.Event) eventHint {
	name, err := filepath.Abs(event.Name)
	if err != nil {
		return eventHint{full: true}
	}
	name = filepath.Clean(name)
	for _, gitRoot := range source.gitRoots {
		if within(gitRoot, name) {
			return eventHint{full: true}
		}
	}
	if !within(source.root, name) || name == source.root {
		return eventHint{full: true}
	}
	relative, err := filepath.Rel(source.root, name)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return eventHint{full: true}
	}
	return eventHint{path: filepath.ToSlash(relative)}
}

func (source *fsNotifySource) emit(hint eventHint) {
	select {
	case source.events <- hint:
	default:
		select {
		case <-source.events:
		default:
		}
		source.events <- eventHint{full: true}
	}
}

func (source *fsNotifySource) Close() error {
	var err error
	source.closeOnce.Do(func() { err = source.watcher.Close() })
	return err
}

func resolveGitControlPaths(root string) ([]string, error) {
	result := make([]string, 0, 2)
	for _, argument := range []string{"--git-dir", "--git-common-dir"} {
		path, err := git(context.Background(), root, "rev-parse", argument)
		if err != nil {
			return nil, err
		}
		if !filepath.IsAbs(path) {
			path = filepath.Join(root, path)
		}
		path, err = filepath.EvalSymlinks(path)
		if err != nil {
			return nil, err
		}
		path, err = filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		path = filepath.Clean(path)
		if !slices.Contains(result, path) {
			result = append(result, path)
		}
	}
	return result, nil
}

func within(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
