// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"html/template"
	"regexp"

	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"
)

var htmlSanitizer = bluemonday.UGCPolicy().AllowAttrs("class").Matching(regexp.MustCompile(`^language-[a-zA-Z0-9]+$`)).OnElements("code")

func markdown(md []byte) template.HTML {
	return template.HTML(htmlSanitizer.SanitizeBytes(blackfriday.MarkdownCommon(md)))
}
