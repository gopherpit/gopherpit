// Copyright (c) 2017 Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package logging

// GRPCLoggerV2 implements methods to satisfy interface
// google.golang.org/grpc/grpclog.LoggerV2.
type GRPCLoggerV2 struct {
	l *Logger
}

// NewGRPCLoggerV2 creates GRPCLoggerV2 that will use
// Logger for logging messages.
//
// Example:
//
//   logger, _ := logging.GetLogger("default")
//   grpclog.SetLoggerV2(logging.NewGRPCLoggerV2(logger))
func NewGRPCLoggerV2(l *Logger) *GRPCLoggerV2 {
	return &GRPCLoggerV2{
		l: l,
	}
}

// Info calls Logger.Info method with provided arguments.
func (g GRPCLoggerV2) Info(args ...interface{}) {
	g.l.Info(args...)
}

// Infoln calls Logger.Info method with provided arguments.
func (g GRPCLoggerV2) Infoln(args ...interface{}) {
	g.l.Info(args...)
}

// Infof calls Logger.Infof method with provided arguments.
func (g GRPCLoggerV2) Infof(format string, args ...interface{}) {
	g.l.Infof(format, args...)
}

// Warning calls Logger.Warning method with provided arguments.
func (g GRPCLoggerV2) Warning(args ...interface{}) {
	g.l.Warning(args...)
}

// Warningln calls Logger.Warning method with provided arguments.
func (g GRPCLoggerV2) Warningln(args ...interface{}) {
	g.l.Warning(args...)
}

// Warningf calls Logger.Warningf method with provided arguments.
func (g GRPCLoggerV2) Warningf(format string, args ...interface{}) {
	g.l.Warningf(format, args...)
}

// Error calls Logger.Error method with provided arguments.
func (g GRPCLoggerV2) Error(args ...interface{}) {
	g.l.Error(args...)
}

// Errorln calls Logger.Error method with provided arguments.
func (g GRPCLoggerV2) Errorln(args ...interface{}) {
	g.l.Error(args...)
}

// Errorf calls Logger.Errorf method with provided arguments.
func (g GRPCLoggerV2) Errorf(format string, args ...interface{}) {
	g.l.Errorf(format, args...)
}

// Fatal calls Logger.Critical method with provided arguments.
func (g GRPCLoggerV2) Fatal(args ...interface{}) {
	g.l.Critical(args...)
}

// Fatalln calls Logger.Critical method with provided arguments.
func (g GRPCLoggerV2) Fatalln(args ...interface{}) {
	g.l.Critical(args...)
}

// Fatalf calls Logger.Criticalf method with provided arguments.
func (g GRPCLoggerV2) Fatalf(format string, args ...interface{}) {
	g.l.Criticalf(format, args...)
}

// V always returns true.
func (g GRPCLoggerV2) V(l int) bool {
	return true
}
