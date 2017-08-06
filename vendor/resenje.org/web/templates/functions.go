// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package templates

import (
	"errors"
	"fmt"
	"html/template"
	"strings"
	"time"
)

// NewContextFunc creates a new function that can be used to store
// and access arbitrary data by keys.
func NewContextFunc(m map[string]interface{}) func(string) interface{} {
	return func(key string) interface{} {
		if value, ok := m[key]; ok {
			return value
		}
		return nil
	}
}

var defaultFunctions = template.FuncMap{
	"safehtml":        safeHTMLFunc,
	"relative_time":   relativeTimeFunc,
	"year_range":      yearRangeFunc,
	"contains_string": containsStringFunc,
	"html_br":         htmlBrFunc,
	"map":             mapFunc,
}

func safeHTMLFunc(text string) template.HTML {
	return template.HTML(text)
}

func relativeTimeFunc(t time.Time) string {
	const day = 24 * time.Hour
	d := time.Since(t)
	switch {
	case d < time.Second:
		return "just now"
	case d < 2*time.Second:
		return "one second ago"
	case d < time.Minute:
		return fmt.Sprintf("%d seconds ago", d/time.Second)
	case d < 2*time.Minute:
		return "one minute ago"
	case d < time.Hour:
		return fmt.Sprintf("%d minutes ago", d/time.Minute)
	case d < 2*time.Hour:
		return "one hour ago"
	case d < day:
		return fmt.Sprintf("%d hours ago", d/time.Hour)
	case d < 2*day:
		return "one day ago"
	}
	return fmt.Sprintf("%d days ago", d/day)
}

func yearRangeFunc(year int) string {
	curYear := time.Now().Year()
	if year >= curYear {
		return fmt.Sprintf("%d", year)
	}
	return fmt.Sprintf("%d - %d", year, curYear)
}

func containsStringFunc(list []string, element, yes, no string) string {
	for _, e := range list {
		if e == element {
			return yes
		}
	}
	return no
}

func htmlBrFunc(text string) string {
	text = template.HTMLEscapeString(text)
	return strings.Replace(text, "\n", "<br>", -1)
}

func mapFunc(values ...interface{}) (map[string]interface{}, error) {
	if len(values)%2 != 0 {
		return nil, errors.New("invalid map call")
	}
	m := make(map[string]interface{}, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, errors.New("map keys must be strings")
		}
		m[key] = values[i+1]
	}
	return m, nil
}
