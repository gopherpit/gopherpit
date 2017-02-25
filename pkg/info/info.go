// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package info // import "gopherpit.com/gopherpit/pkg/info"

import (
	"os"
	"runtime"
	"runtime/pprof"

	"resenje.org/marshal"
)

// Information represents selected data provided by memstats and pprof.
type Information struct {
	// Server's hostname.
	Host string `json:"host"`
	// Server's operating system.
	OS string `json:"os"`
	// Server's architecture.
	Arch string `json:"arch"`
	// Number of server's CPUs
	NumCPU int `json:"cpu-count"`
	// Version of Go that executable was build with.
	GoVersion string `json:"go-version"`

	// General statistics.

	// Bytes allocated and not yet freed.
	Alloc marshal.Bytes `json:"mem-used"`
	// Bytes allocated and not yet freed.
	AllocBytes uint64 `json:"mem-used-bytes"`
	// Bytes allocated (even if freed).
	TotalAlloc marshal.Bytes `json:"mem-allocated"`
	// Bytes allocated (even if freed).
	TotalAllocBytes uint64 `json:"mem-allocated-bytes"`
	// Bytes obtained from system.
	Sys marshal.Bytes `json:"mem-sys"`
	// Bytes obtained from system.
	SysBytes uint64 `json:"mem-sys-bytes"`
	// Number of pointer lookups.
	Lookups uint64 `json:"lookups"`
	// Number of mallocs.
	Mallocs uint64 `json:"mallocs"`
	// Number of frees.
	Frees uint64 `json:"mem-frees"`

	// Main allocation heap statistics.

	// Bytes allocated and not yet freed (same as Alloc above).
	HeapAlloc marshal.Bytes `json:"heap-used"`
	// Bytes allocated and not yet freed (same as Alloc above).
	HeapAllocBytes uint64 `json:"heap-used-bytes"`
	// Bytes obtained from system.
	HeapSys marshal.Bytes `json:"heap-sys"`
	// Bytes obtained from system.
	HeapSysBytes uint64 `json:"heap-sys-bytes"`
	// Bytes in idle spans.
	HeapIdle marshal.Bytes `json:"heap-idle"`
	// Bytes in idle spans.
	HeapIdleBytes uint64 `json:"heap-idle-bytes"`
	// Bytes in non-idle span.
	HeapInuse marshal.Bytes `json:"heap-inuse"`
	// Bytes in non-idle span.
	HeapInuseBytes uint64 `json:"heap-inuse-bytes"`
	// Bytes released to the OS.
	HeapReleased marshal.Bytes `json:"heap-released"`
	// Bytes released to the OS.
	HeapReleasedBytes uint64 `json:"heap-released-bytes"`
	// Total number of allocated objects.
	HeapObjects uint64 `json:"heap-objects"`

	// Pprof stack counts
	StackCount map[string]int `json:"stack-count"`
}

// NewInformation creates and populates a new instance of info.Information.
func NewInformation() *Information {
	i := &Information{}
	i.Update()
	return i
}

// Update updates Information with current data.
func (i *Information) Update() {
	host, err := os.Hostname()
	if err != nil {
		host = ""
	}
	memStats := runtime.MemStats{}
	runtime.ReadMemStats(&memStats)
	stackCount := map[string]int{}
	for _, p := range pprof.Profiles() {
		stackCount[p.Name()] = p.Count()
	}
	i.Host = host
	i.OS = runtime.GOOS
	i.Arch = runtime.GOARCH
	i.NumCPU = runtime.NumCPU()
	i.GoVersion = runtime.Version()
	i.Alloc = marshal.Bytes(memStats.Alloc)
	i.AllocBytes = memStats.Alloc
	i.TotalAlloc = marshal.Bytes(memStats.TotalAlloc)
	i.TotalAllocBytes = memStats.TotalAlloc
	i.Sys = marshal.Bytes(memStats.Sys)
	i.SysBytes = memStats.Sys
	i.Lookups = memStats.Lookups
	i.Mallocs = memStats.Mallocs
	i.Frees = memStats.Frees
	i.HeapAlloc = marshal.Bytes(memStats.HeapAlloc)
	i.HeapAllocBytes = memStats.HeapAlloc
	i.HeapSys = marshal.Bytes(memStats.HeapSys)
	i.HeapSysBytes = memStats.HeapSys
	i.HeapIdle = marshal.Bytes(memStats.HeapIdle)
	i.HeapIdleBytes = memStats.HeapIdle
	i.HeapInuse = marshal.Bytes(memStats.HeapInuse)
	i.HeapInuseBytes = memStats.HeapInuse
	i.HeapReleased = marshal.Bytes(memStats.HeapReleased)
	i.HeapReleasedBytes = memStats.HeapReleased
	i.HeapObjects = memStats.HeapObjects
	i.StackCount = stackCount
}
