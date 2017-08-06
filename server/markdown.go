// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"
)

var htmlSanitizer = bluemonday.UGCPolicy().AllowAttrs("class").Globally()

func markdown(md []byte) template.HTML {
	return template.HTML(blackfriday.MarkdownCommon(md))
}

func markdownSanitized(md []byte) template.HTML {
	return template.HTML(htmlSanitizer.SanitizeBytes(blackfriday.MarkdownCommon(md)))
}

func parseMarkdown(dir string) (fragments map[string]interface{}, err error) {
	fragments = map[string]interface{}{}
	_, err = os.Stat(dir)
	switch {
	case os.IsNotExist(err):
	case err == nil:
		if err = filepath.Walk(dir, func(path string, _ os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !strings.HasSuffix(path, ".md") {
				return nil
			}
			data, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			name := strings.TrimPrefix(path, dir+"/")
			name = strings.TrimSuffix(name, ".md")
			fragments[name] = markdown(data)
			return nil
		}); err != nil {
			return
		}
	}
	return
}
