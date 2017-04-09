// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gcrastore // import "gopherpit.com/gopherpit/services/gcrastore"

import throttled "gopkg.in/throttled/throttled.v2"

// Service encapsulates throttled.GCRAStore interface to be used as
// rate limiter store.
type Service interface {
	throttled.GCRAStore
}
