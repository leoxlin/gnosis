package vault

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/adrg/frontmatter"
)

// WriteDocument writes content into the local vault configured directly under
// root. The target URI must identify a page in that local vault.
func WriteDocument(root, uri string, content []byte, update bool) (string, error) {
	fields := frontmatterFields{}
	_, err := frontmatter.MustParse(strings.NewReader(string(content)), &fields, yamlFrontmatter)
	if err != nil {
		return "", fmt.Errorf("write: parse input: %w", frontmatterError(err))
	}
	inputType, err := requiredSearchScalar(fields, "type")
	if err != nil {
		return "", fmt.Errorf("write: input %w", err)
	}
	if _, err := requiredSearchScalar(fields, "title"); err != nil {
		return "", fmt.Errorf("write: input %w", err)
	}

	resolution, err := ResolveConfig(root)
	if err != nil {
		return "", fmt.Errorf("write: resolve current directory: %w", err)
	}
	if len(resolution.LocalVaultRoots) == 0 {
		return "", fmt.Errorf("write: no local vault is defined in the current directory")
	}
	if len(resolution.LocalVaultRoots) != 1 {
		return "", fmt.Errorf("write: current directory resolves multiple local vaults")
	}
	localRoot := filepath.Clean(resolution.LocalVaultRoots[0])
	destination, destinationPath, err := writeURIDestination(uri, resolution.Config.Vault.Name, localRoot)
	if err != nil {
		return "", fmt.Errorf("write: target URI: %w", err)
	}

	source, err := NewSearchSource(root)
	if err != nil {
		return "", fmt.Errorf("write: %w", err)
	}
	pages, err := source.pages()
	if err != nil {
		return "", fmt.Errorf("write: %w", err)
	}

	conceptPage, err := conceptTypePage(pages, inputType)
	if err != nil {
		return "", err
	}
	conceptFields := frontmatterFields{}
	_, err = frontmatter.MustParse(strings.NewReader(string(conceptPage.data)), &conceptFields, yamlFrontmatter)
	if err != nil {
		return "", fmt.Errorf("write: parse Concept Type %q: %w", inputType, frontmatterError(err))
	}
	directory, err := requiredSearchScalar(conceptFields, "path")
	if err != nil {
		return "", fmt.Errorf("write: Concept Type %q %w", inputType, err)
	}
	destinationDirectory, err := writeDestinationDirectory(localRoot, directory)
	if err != nil {
		return "", fmt.Errorf("write: Concept Type %q path: %w", inputType, err)
	}
	relative, err := filepath.Rel(destinationDirectory, destination)
	if err != nil {
		return "", fmt.Errorf("write: determine target path: %w", err)
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("write: target path is outside Concept Type %q path", inputType)
	}
	if !update && hasExternalCollision(pages, localRoot, destinationPath) {
		return "", fmt.Errorf("write: document already exists outside the current vault; rerun with --update")
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return "", err
	}
	if err := atomicWriteFile(destination, content, 0o644); err != nil {
		return "", err
	}
	return destination, nil
}

func writeURIDestination(rawURI, vaultName, localRoot string) (string, string, error) {
	u, err := url.Parse(strings.TrimSpace(rawURI))
	if err != nil || u.Scheme != "gnosis" || u.Host == "" || u.RawQuery != "" || u.Fragment != "" {
		return "", "", fmt.Errorf("must be a canonical gnosis URI")
	}
	if u.Host != vaultName {
		return "", "", fmt.Errorf("vault %q is not the current local vault %q", u.Host, vaultName)
	}
	escapedPath := strings.TrimPrefix(u.EscapedPath(), "/")
	decodedPath, err := url.PathUnescape(escapedPath)
	if err != nil || decodedPath == "" || strings.Contains(decodedPath, "\\") || path.IsAbs(decodedPath) || path.Clean(decodedPath) != decodedPath || strings.HasPrefix(decodedPath, "../") {
		return "", "", fmt.Errorf("path must be a canonical vault-relative document path")
	}
	if documentURI(vaultName, decodedPath) != rawURI {
		return "", "", fmt.Errorf("must be canonical")
	}
	destination := filepath.Clean(filepath.Join(localRoot, filepath.FromSlash(decodedPath)))
	relative, err := filepath.Rel(localRoot, destination)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("path escapes the local vault")
	}
	return destination, filepath.ToSlash(relative), nil
}

func conceptTypePage(pages []*searchPage, title string) (*searchPage, error) {
	matches := make([]*searchPage, 0, 1)
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

func hasExternalCollision(pages []*searchPage, localRoot, destinationPath string) bool {
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
