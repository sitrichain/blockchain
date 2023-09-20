/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package log

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewRaftLogger creates a logger that delegates to the zap.SugaredLogger.
func NewRaftLogger(l *zap.SugaredLogger) *RaftLogger {
	return &RaftLogger{
		s: l,
	}
}

// A RaftLogger is an adapter around a zap.SugaredLogger that provides
// structured logging capabilities while preserving much of the legacy logging
// behavior.
//
// The most significant difference between the RaftLogger and the
// zap.SugaredLogger is that methods without a formatting suffix (f or w) build
// the log entry message with fmt.Sprintln instead of fmt.Sprint. Without this
// change, arguments are not separated by spaces.
type RaftLogger struct{ s *zap.SugaredLogger }

func (f *RaftLogger) DPanic(args ...interface{})                    { f.s.DPanicf(formatArgs(args)) }
func (f *RaftLogger) DPanicf(template string, args ...interface{})  { f.s.DPanicf(template, args...) }
func (f *RaftLogger) DPanicw(msg string, kvPairs ...interface{})    { f.s.DPanicw(msg, kvPairs...) }
func (f *RaftLogger) Debug(args ...interface{})                     { f.s.Debugf(formatArgs(args)) }
func (f *RaftLogger) Debugf(template string, args ...interface{})   { f.s.Debugf(template, args...) }
func (f *RaftLogger) Debugw(msg string, kvPairs ...interface{})     { f.s.Debugw(msg, kvPairs...) }
func (f *RaftLogger) Error(args ...interface{})                     { f.s.Errorf(formatArgs(args)) }
func (f *RaftLogger) Errorf(template string, args ...interface{})   { f.s.Errorf(template, args...) }
func (f *RaftLogger) Errorw(msg string, kvPairs ...interface{})     { f.s.Errorw(msg, kvPairs...) }
func (f *RaftLogger) Fatal(args ...interface{})                     { f.s.Fatalf(formatArgs(args)) }
func (f *RaftLogger) Fatalf(template string, args ...interface{})   { f.s.Fatalf(template, args...) }
func (f *RaftLogger) Fatalw(msg string, kvPairs ...interface{})     { f.s.Fatalw(msg, kvPairs...) }
func (f *RaftLogger) Info(args ...interface{})                      { f.s.Infof(formatArgs(args)) }
func (f *RaftLogger) Infof(template string, args ...interface{})    { f.s.Infof(template, args...) }
func (f *RaftLogger) Infow(msg string, kvPairs ...interface{})      { f.s.Infow(msg, kvPairs...) }
func (f *RaftLogger) Panic(args ...interface{})                     { f.s.Panicf(formatArgs(args)) }
func (f *RaftLogger) Panicf(template string, args ...interface{})   { f.s.Panicf(template, args...) }
func (f *RaftLogger) Panicw(msg string, kvPairs ...interface{})     { f.s.Panicw(msg, kvPairs...) }
func (f *RaftLogger) Warn(args ...interface{})                      { f.s.Warnf(formatArgs(args)) }
func (f *RaftLogger) Warnf(template string, args ...interface{})    { f.s.Warnf(template, args...) }
func (f *RaftLogger) Warnw(msg string, kvPairs ...interface{})      { f.s.Warnw(msg, kvPairs...) }
func (f *RaftLogger) Warning(args ...interface{})                   { f.s.Warnf(formatArgs(args)) }
func (f *RaftLogger) Warningf(template string, args ...interface{}) { f.s.Warnf(template, args...) }

// for backwards compatibility
func (f *RaftLogger) Critical(args ...interface{})                   { f.s.Errorf(formatArgs(args)) }
func (f *RaftLogger) Criticalf(template string, args ...interface{}) { f.s.Errorf(template, args...) }
func (f *RaftLogger) Notice(args ...interface{})                     { f.s.Infof(formatArgs(args)) }
func (f *RaftLogger) Noticef(template string, args ...interface{})   { f.s.Infof(template, args...) }

func (f *RaftLogger) Named(name string) *RaftLogger { return &RaftLogger{s: f.s.Named(name)} }
func (f *RaftLogger) Sync() error                   { return f.s.Sync() }
func (f *RaftLogger) Zap() *zap.Logger              { return f.s.Desugar() }

func (f *RaftLogger) IsEnabledFor(level zapcore.Level) bool {
	return f.s.Desugar().Core().Enabled(level)
}

func (f *RaftLogger) With(args ...interface{}) *RaftLogger {
	return &RaftLogger{s: f.s.With(args...)}
}

func (f *RaftLogger) WithOptions(opts ...zap.Option) *RaftLogger {
	l := f.s.Desugar().WithOptions(opts...)
	return &RaftLogger{s: l.Sugar()}
}

func formatArgs(args []interface{}) string { return strings.TrimSuffix(fmt.Sprintln(args...), "\n") }
