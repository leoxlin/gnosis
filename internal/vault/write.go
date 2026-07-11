package vault

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

// WriteDocument writes content into the local vault configured directly under
// root. Its Concept Type determines the vault-relative destination directory.
func WriteDocument(root, conceptType, title string, content []byte, overwrite bool) (string, error) {
	conceptType = strings.TrimSpace(conceptType)
	if conceptType == "" {
		return "", fmt.Errorf("write: -type must not be empty")
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return "", fmt.Errorf("write: -title must not be empty")
	}

	fields, _, err := parseFrontmatter(string(content))
	if err != nil {
		return "", fmt.Errorf("write: parse input: %w", err)
	}
	inputType, err := requiredSearchScalar(fields, "type")
	if err != nil {
		return "", fmt.Errorf("write: input %w", err)
	}
	if inputType != conceptType {
		return "", fmt.Errorf("write: input frontmatter type %q does not match -type %q", inputType, conceptType)
	}
	inputTitle, err := requiredSearchScalar(fields, "title")
	if err != nil {
		return "", fmt.Errorf("write: input %w", err)
	}
	if inputTitle != title {
		return "", fmt.Errorf("write: input frontmatter title %q does not match -title %q", inputTitle, title)
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

	source, err := NewSearchSource(root)
	if err != nil {
		return "", fmt.Errorf("write: %w", err)
	}
	pages, err := source.pages()
	if err != nil {
		return "", fmt.Errorf("write: %w", err)
	}

	conceptPage, err := conceptTypePage(pages, conceptType)
	if err != nil {
		return "", err
	}
	conceptFields, _, err := parseFrontmatter(string(conceptPage.data))
	if err != nil {
		return "", fmt.Errorf("write: parse Concept Type %q: %w", conceptType, err)
	}
	directory, err := requiredSearchScalar(conceptFields, "path")
	if err != nil {
		return "", fmt.Errorf("write: Concept Type %q %w", conceptType, err)
	}
	destinationDirectory, err := writeDestinationDirectory(localRoot, directory)
	if err != nil {
		return "", fmt.Errorf("write: Concept Type %q path: %w", conceptType, err)
	}
	destination, err := localIdentityDestination(pages, localRoot, directory, conceptType, title)
	if err != nil {
		return "", err
	}
	if destination == "" && overwrite {
		destination, err = externalIdentityDestination(pages, localRoot, directory, conceptType, title)
		if err != nil {
			return "", err
		}
	}
	if destination == "" {
		filename := slugify(title)
		if filename == "" {
			return "", fmt.Errorf("write: -title %q cannot produce a filename", title)
		}
		destination = filepath.Join(destinationDirectory, filename+".md")
	}
	destinationID, err := filepath.Rel(localRoot, destination)
	if err != nil {
		return "", fmt.Errorf("write: determine destination: %w", err)
	}
	destinationID = filepath.ToSlash(destinationID)

	if !overwrite && hasExternalCollision(pages, localRoot, destinationID, conceptType, title) {
		return "", fmt.Errorf("write: document already exists outside the current vault; rerun with -overwrite")
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return "", err
	}
	if err := atomicWriteFile(destination, content, 0o644); err != nil {
		return "", err
	}
	return destination, nil
}

func externalIdentityDestination(pages []*searchPage, localRoot, directory, conceptType, title string) (string, error) {
	matches := make([]string, 0, 1)
	for _, page := range pages {
		if page.root == localRoot || page.document.Type != conceptType || page.document.Title != title {
			continue
		}
		relative, err := filepath.Rel(filepath.Clean(directory), filepath.FromSlash(page.document.ID))
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			continue
		}
		matches = append(matches, filepath.Join(localRoot, filepath.FromSlash(page.document.ID)))
	}
	switch len(matches) {
	case 0:
		return "", nil
	case 1:
		return matches[0], nil
	default:
		sort.Strings(matches)
		return "", fmt.Errorf("write: multiple external documents found with type %q and title %q: %s", conceptType, title, strings.Join(matches, ", "))
	}
}

func localIdentityDestination(pages []*searchPage, localRoot, directory, conceptType, title string) (string, error) {
	matches := make([]string, 0, 1)
	for _, page := range pages {
		if page.root != localRoot || page.document.Type != conceptType || page.document.Title != title {
			continue
		}
		relative, err := filepath.Rel(filepath.Clean(directory), filepath.FromSlash(page.document.ID))
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			continue
		}
		matches = append(matches, filepath.Join(localRoot, filepath.FromSlash(page.document.ID)))
	}
	switch len(matches) {
	case 0:
		return "", nil
	case 1:
		return matches[0], nil
	default:
		sort.Strings(matches)
		return "", fmt.Errorf("write: multiple local documents found with type %q and title %q: %s", conceptType, title, strings.Join(matches, ", "))
	}
}

func conceptTypePage(pages []*searchPage, title string) (*searchPage, error) {
	matches := make([]*searchPage, 0, 1)
	for _, page := range pages {
		if page.document.Type == "Concept Type" && page.document.Title == title {
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

func hasExternalCollision(pages []*searchPage, localRoot, destinationID, conceptType, title string) bool {
	for _, page := range pages {
		if page.root == localRoot {
			continue
		}
		if page.document.ID == destinationID || (page.document.Type == conceptType && page.document.Title == title) {
			return true
		}
	}
	return false
}

func slugify(value string) string {
	var result strings.Builder
	separator := false
	for _, character := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(character) || unicode.IsDigit(character) {
			if separator && result.Len() > 0 {
				result.WriteByte('-')
			}
			result.WriteRune(character)
			separator = false
			continue
		}
		if result.Len() > 0 {
			separator = true
		}
	}
	return result.String()
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
