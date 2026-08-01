//go:build linux && amd64

package languagepack

/*
#cgo LDFLAGS: -ldl
#include <dlfcn.h>
#include <stdlib.h>

typedef const void *(*gnosis_language_fn)(void);

static void *gnosis_dlopen(const char *path) { return dlopen(path, RTLD_NOW | RTLD_LOCAL); }
static const void *gnosis_dlsym_language(void *handle, const char *symbol) {
	gnosis_language_fn fn = (gnosis_language_fn)dlsym(handle, symbol);
	return fn == NULL ? NULL : fn();
}
static const char *gnosis_dlerror(void) { return dlerror(); }
static int gnosis_dlclose(void *handle) { return dlclose(handle); }
*/
import "C"

import (
	"fmt"
	"strings"
	"unsafe"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

type nativeLanguage struct {
	handle   unsafe.Pointer
	language *tree_sitter.Language
}

func openNativeLanguage(path, name string) (nativeLanguage, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))
	handle := C.gnosis_dlopen(cPath)
	if handle == nil {
		return nativeLanguage{}, fmt.Errorf("load parser library %q: %s", path, C.GoString(C.gnosis_dlerror()))
	}
	symbol := C.CString("tree_sitter_" + strings.ReplaceAll(name, "-", "_"))
	defer C.free(unsafe.Pointer(symbol))
	language := C.gnosis_dlsym_language(handle, symbol)
	if language == nil {
		C.gnosis_dlclose(handle)
		return nativeLanguage{}, fmt.Errorf("parser library %q has no language symbol", path)
	}
	return nativeLanguage{handle: handle, language: tree_sitter.NewLanguage(unsafe.Pointer(language))}, nil
}

func (language nativeLanguage) close() { C.gnosis_dlclose(language.handle) }
