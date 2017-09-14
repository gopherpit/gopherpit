// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package templates

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
)

// Error is a common error type that holds
// information about error message and template name.
type Error struct {
	Err      error
	Template string
}

func (e *Error) Error() string {
	if e.Template == "" {
		return e.Err.Error()
	}
	return fmt.Sprintf("%s: %s", e.Err.Error(), e.Template)
}

// FileReadFunc returns the content of file referenced
// by filename. It hes the same signature as ioutil.ReadFile
// function.
type FileReadFunc func(filename string) ([]byte, error)

// ErrUnknownTemplate will be returned by Render function if
// the template does not exist.
var ErrUnknownTemplate = fmt.Errorf("unknown template")

// Options holds parameters for creating Templates.
type Options struct {
	fileFindFunc func(filename string) string
	fileReadFunc FileReadFunc
	contentType  string
	files        map[string][]string
	strings      map[string][]string
	functions    template.FuncMap
	delimOpen    string
	delimClose   string
	logf         func(format string, a ...interface{})
}

// Option sets parameters used in New function.
type Option func(*Options)

// WithContentType sets the content type HTTP header that
// will be written on Render and Response functions.
func WithContentType(contentType string) Option {
	return func(o *Options) { o.contentType = contentType }
}

// WithBaseDir sets the directory in which template files
// are stored.
func WithBaseDir(dir string) Option {
	return func(o *Options) {
		o.fileFindFunc = func(f string) string {
			return filepath.Join(dir, f)
		}
	}
}

// WithFileFindFunc sets the function that will return the
// file path on disk based on filename provided from files
// defind using WithTemplateFromFile or WithTemplateFromFiles.
func WithFileFindFunc(fn func(filename string) string) Option {
	return func(o *Options) { o.fileFindFunc = fn }
}

// WithFileReadFunc sets the function that will return the
// content of template given the filename.
func WithFileReadFunc(fn FileReadFunc) Option {
	return func(o *Options) { o.fileReadFunc = fn }
}

// WithTemplateFromFiles adds a template parsed from files.
func WithTemplateFromFiles(name string, files ...string) Option {
	return func(o *Options) { o.files[name] = files }
}

// WithTemplatesFromFiles adds a map of templates parsed from files.
func WithTemplatesFromFiles(ts map[string][]string) Option {
	return func(o *Options) {
		for name, files := range ts {
			o.files[name] = files
		}
	}
}

// WithTemplateFromStrings adds a template parsed from string.
func WithTemplateFromStrings(name string, strings ...string) Option {
	return func(o *Options) { o.strings[name] = strings }
}

// WithTemplatesFromStrings adds a map of templates parsed from strings.
func WithTemplatesFromStrings(ts map[string][]string) Option {
	return func(o *Options) {
		for name, strings := range ts {
			o.strings[name] = strings
		}
	}
}

// WithFunction adds a function to templates.
func WithFunction(name string, fn interface{}) Option {
	return func(o *Options) { o.functions[name] = fn }
}

// WithFunctions adds function map to templates.
func WithFunctions(fns template.FuncMap) Option {
	return func(o *Options) {
		for name, fn := range fns {
			o.functions[name] = fn
		}
	}
}

// WithDelims sets the delimiters used in templates.
func WithDelims(open, close string) Option {
	return func(o *Options) {
		o.delimOpen = open
		o.delimClose = close
	}
}

// WithLogFunc sets the function that will perform message logging.
// Default is log.Printf.
func WithLogFunc(logf func(format string, a ...interface{})) Option {
	return func(o *Options) { o.logf = logf }
}

// Templates structure holds parsed templates.
type Templates struct {
	templates   map[string]*template.Template
	defaultName string
	contentType string
	logf        func(format string, a ...interface{})
}

// New creates a new instance of Templates and parses
// provided files and strings.
func New(opts ...Option) (t *Templates, err error) {
	functions := template.FuncMap{}
	for name, fn := range defaultFunctions {
		functions[name] = fn
	}
	o := &Options{
		fileFindFunc: func(f string) string {
			return f
		},
		fileReadFunc: ioutil.ReadFile,
		files:        map[string][]string{},
		functions:    functions,
		delimOpen:    "{{",
		delimClose:   "}}",
		logf:         log.Printf,
	}
	for _, opt := range opts {
		opt(o)
	}

	t = &Templates{
		templates:   map[string]*template.Template{},
		contentType: o.contentType,
		logf:        o.logf,
	}
	for name, strings := range o.strings {
		tpl, err := parseStrings(template.New("").Funcs(o.functions).Delims(o.delimOpen, o.delimClose), strings...)
		if err != nil {
			return nil, err
		}
		t.templates[name] = tpl
	}
	for name, files := range o.files {
		fs := []string{}
		for _, f := range files {
			fs = append(fs, o.fileFindFunc(f))
		}
		tpl, err := parseFiles(o.fileReadFunc, template.New("").Funcs(o.functions).Delims(o.delimOpen, o.delimClose), fs...)
		if err != nil {
			return nil, err
		}
		t.templates[name] = tpl
	}
	return
}

// RespondTemplateWithStatus executes a named template with provided data into buffer,
// then writes the the status and body to the response writer.
// A panic will be raised if the template does not exist or fails to execute.
func (t Templates) RespondTemplateWithStatus(w http.ResponseWriter, name, templateName string, data interface{}, status int) {
	buf := bytes.Buffer{}
	tpl, ok := t.templates[name]
	if !ok {
		panic(&Error{Err: ErrUnknownTemplate, Template: name})
	}
	if err := tpl.ExecuteTemplate(&buf, templateName, data); err != nil {
		panic(err)
	}
	if t.contentType != "" {
		w.Header().Set("Content-Type", t.contentType)
	}
	if status > 0 {
		w.WriteHeader(status)
	}
	if _, err := buf.WriteTo(w); err != nil {
		t.logf("respond %q template %q: %v", name, templateName, err)
	}
}

// RespondWithStatus executes a template with provided data into buffer,
// then writes the the status and body to the response writer.
// A panic will be raised if the template does not exist or fails to execute.
func (t Templates) RespondWithStatus(w http.ResponseWriter, name string, data interface{}, status int) {
	buf := bytes.Buffer{}
	tpl, ok := t.templates[name]
	if !ok {
		panic(&Error{Err: ErrUnknownTemplate, Template: name})
	}
	if err := tpl.Execute(&buf, data); err != nil {
		panic(err)
	}
	if t.contentType != "" {
		w.Header().Set("Content-Type", t.contentType)
	}
	if status > 0 {
		w.WriteHeader(status)
	}
	if _, err := buf.WriteTo(w); err != nil {
		t.logf("respond %q: %v", name, err)
	}
}

// RespondTemplate executes a named template with provided data into buffer,
// then writes the the body to the response writer.
// A panic will be raised if the template does not exist or fails to execute.
func (t Templates) RespondTemplate(w http.ResponseWriter, name, templateName string, data interface{}) {
	t.RespondTemplateWithStatus(w, name, templateName, data, 0)
}

// Respond executes template with provided data into buffer,
// then writes the the body to the response writer.
// A panic will be raised if the template does not exist or fails to execute.
func (t Templates) Respond(w http.ResponseWriter, name string, data interface{}) {
	t.RespondWithStatus(w, name, data, 0)
}

// RenderTemplate executes a named template and returns the string.
func (t Templates) RenderTemplate(name, templateName string, data interface{}) (s string, err error) {
	buf := bytes.Buffer{}
	tpl, ok := t.templates[name]
	if !ok {
		return "", &Error{Err: ErrUnknownTemplate, Template: name}
	}
	if err := tpl.ExecuteTemplate(&buf, templateName, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// Render executes a template and returns the string.
func (t Templates) Render(name string, data interface{}) (s string, err error) {
	buf := bytes.Buffer{}
	tpl, ok := t.templates[name]
	if !ok {
		return "", &Error{Err: ErrUnknownTemplate, Template: name}
	}
	if err := tpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func parseFiles(fn FileReadFunc, t *template.Template, filenames ...string) (*template.Template, error) {
	for _, filename := range filenames {
		b, err := fn(filename)
		if err != nil {
			return nil, err
		}
		_, err = t.Parse(string(b))
		if err != nil {
			return nil, err
		}
	}
	return t, nil
}

func parseStrings(t *template.Template, strings ...string) (*template.Template, error) {
	for _, str := range strings {
		_, err := t.Parse(str)
		if err != nil {
			return nil, err
		}
	}
	return t, nil
}
