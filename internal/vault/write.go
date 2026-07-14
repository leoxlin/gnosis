package vault

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WriteDocument writes content into the local vault configured directly under
// root. The target URI must identify a page in that local vault.
func WriteDocument(root, uri string, content []byte, update bool) (string, error) {
	parsed, err := parsePage(content)
	if err != nil {
		return "", fmt.Errorf("write: parse input: %w", err)
	}
	metadata, problems := interpretPageMetadata(parsed.fields)
	if len(problems) > 0 {
		return "", fmt.Errorf("write: input %w", problems[0])
	}
	if _, err := requiredPageScalar(parsed.fields, "title"); err != nil {
		return "", fmt.Errorf("write: input %w", err)
	}
	vault, err := loadEffectiveVault(root)
	if err != nil {
		return "", fmt.Errorf("write: resolve current directory: %w", err)
	}
	localRoot, hasLocalRoot := vault.localRoot()
	if !hasLocalRoot {
		return "", fmt.Errorf("write: no local vault is defined in the current directory")
	}
	localRoot = filepath.Clean(localRoot)
	destination, destinationPath, err := writeURIDestination(uri, vault.config.Vault.Name, localRoot)
	if err != nil {
		return "", fmt.Errorf("write: target URI: %w", err)
	}

	pages, err := vault.pages()
	if err != nil {
		return "", fmt.Errorf("write: %w", err)
	}

	conceptPage, err := conceptTypePage(pages, metadata.conceptType)
	if err != nil {
		return "", err
	}
	directory, err := requiredPageScalar(conceptPage.fields, "path")
	if err != nil {
		return "", fmt.Errorf("write: Concept Type %q %w", metadata.conceptType, err)
	}
	destinationDirectory, err := writeDestinationDirectory(localRoot, directory)
	if err != nil {
		return "", fmt.Errorf("write: Concept Type %q path: %w", metadata.conceptType, err)
	}
	relative, err := filepath.Rel(destinationDirectory, destination)
	if err != nil {
		return "", fmt.Errorf("write: determine target path: %w", err)
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("write: target path is outside Concept Type %q path", metadata.conceptType)
	}
	if !update && hasExternalCollision(pages, localRoot, destinationPath) {
		return "", fmt.Errorf("write: document already exists outside the current vault; rerun with --update")
	}
	candidate, err := buildEffectivePage(localRoot, destination, content, Origin{
		Vault:      vault.config.Vault.Name,
		Kind:       OriginLocal,
		Root:       localRoot,
		Path:       destination,
		Precedence: 0,
	}, parsed, metadata)
	if err != nil {
		return "", fmt.Errorf("write: build input page: %w", err)
	}
	resolverPages := append([]*effectivePage(nil), pages...)
	resolverPages = append(resolverPages, candidate)
	if _, err := newDocumentResolver(resolverPages).resolvePageLinks(candidate); err != nil {
		return "", fmt.Errorf("write: input links: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return "", err
	}
	if err := atomicWriteFile(destination, content, 0o644); err != nil {
		return "", err
	}
	if err := vault.publish("gnosis: update vault"); err != nil {
		return "", fmt.Errorf("write: publish backend: %w", err)
	}
	return destination, nil
}

func writeURIDestination(rawURI, vaultName, localRoot string) (string, string, error) {
	targetVault, decodedPath, ok := canonicalGnosisParts(rawURI)
	if !ok {
		return "", "", fmt.Errorf("must be a canonical gnosis URI")
	}
	if targetVault != vaultName {
		return "", "", fmt.Errorf("vault %q is not the current local vault %q", targetVault, vaultName)
	}
	if filepath.Ext(decodedPath) != ".md" {
		return "", "", fmt.Errorf("path must end in lowercase .md")
	}
	if reservedPageName(filepath.Base(decodedPath)) {
		return "", "", fmt.Errorf("path must not use reserved name %q", filepath.Base(decodedPath))
	}
	destination := filepath.Clean(filepath.Join(localRoot, filepath.FromSlash(decodedPath)))
	relative, err := filepath.Rel(localRoot, destination)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("path escapes the local vault")
	}
	return destination, filepath.ToSlash(relative), nil
}

func conceptTypePage(pages []*effectivePage, title string) (*effectivePage, error) {
	matches := make([]*effectivePage, 0, 1)
	for _, page := range pages {
		if page.document.Type == "ConceptType" && page.document.Title == title {
			matches = append(matches, page)
		}
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("write: no Concept Type found with title %q", title)
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf("write: multiple Concept Types found with title %q", title)
	}
}

func writeDestinationDirectory(localRoot, path string) (string, error) {
	if filepath.IsAbs(path) {
		return "", fmt.Errorf("must be vault-relative")
	}
	destination := filepath.Clean(filepath.Join(localRoot, path))
	relative, err := filepath.Rel(localRoot, destination)
	if err != nil {
		return "", err
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("escapes the local vault")
	}
	return destination, nil
}

func hasExternalCollision(pages []*effectivePage, localRoot, destinationPath string) bool {
	for _, page := range pages {
		if page.root == localRoot {
			continue
		}
		if page.document.Path == destinationPath {
			return true
		}
	}
	return false
}

// WriteGeneratedFile writes content to path atomically, skipping the write when
// the existing file already holds identical content. When overwrite is false an
// existing file is always left untouched. It reports whether the file changed.
func WriteGeneratedFile(path string, content []byte, overwrite bool) (bool, error) {
	existing, err := os.ReadFile(path)
	switch {
	case err == nil:
		if !overwrite || bytes.Equal(existing, content) {
			return false, nil
		}
	case !os.IsNotExist(err):
		return false, err
	}

	if err := atomicWriteFile(path, content, 0o644); err != nil {
		return false, err
	}
	return true, nil
}

func atomicWriteFile(path string, content []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer func() {
		_ = temp.Close()
		_ = os.Remove(tempPath)
	}()

	if err := temp.Chmod(mode); err != nil {
		return err
	}
	if _, err := temp.Write(content); err != nil {
		return err
	}
	if err := temp.Sync(); err != nil {
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("replace %s: %w", path, err)
	}
	return nil
}
