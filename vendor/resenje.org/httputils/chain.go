// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httputils // import "resenje.org/httputils"

import "net/http"

// ChainHandlers executes each function from the arguments with handler
// from the next function to construct a chan fo callers.
func ChainHandlers(handlers ...func(http.Handler) http.Handler) (handler http.Handler) {
	for i := len(handlers) - 1; i >= 0; i-- {
		handler = handlers[i](handler)
	}
	return
}
