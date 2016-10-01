package fileServer

import (
	"errors"
	"net/http"
)

var (
	DefaultNotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "Not Found", http.StatusNotFound)
	})
	DefaultForbiddenHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "Forbidden", http.StatusForbidden)
	})
	DefaultInternalServerErrorhandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	})

	errNotFound       = errors.New("not found")
	errNotRegularFile = errors.New("not a regular file")
)

type Options struct {
	Hasher                Hasher
	NoHashQueryStrings    bool
	RedirectTrailingSlash bool
	IndexPage             string

	NotFoundHandler            http.Handler
	ForbiddenHandler           http.Handler
	InternalServerErrorHandler http.Handler
}
