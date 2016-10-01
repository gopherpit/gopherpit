package fileServer

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func redirect(w http.ResponseWriter, r *http.Request, location string) {
	if q := r.URL.RawQuery; q != "" {
		location += "?" + q
	}
	w.Header().Set("Location", location)
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusFound)
}

func open(root, name string) (http.File, error) {
	if filepath.Separator != '/' && strings.IndexRune(name, filepath.Separator) >= 0 ||
		strings.Contains(name, "\x00") {
		return nil, errNotFound // invalid character in file path
	}
	if root == "" {
		root = "."
	}
	f, err := os.Open(filepath.Join(root, filepath.FromSlash(path.Clean("/"+name))))
	if err != nil {
		return nil, err
	}
	return f, nil
}
