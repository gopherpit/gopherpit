// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"html/template"

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
