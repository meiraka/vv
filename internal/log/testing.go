package log

import (
	"fmt"
	"testing"
)

type TestLogger struct {
	tb testing.TB
}

func NewTestLogger(tb testing.TB) *TestLogger {
	return &TestLogger{tb}
}

func (l *TestLogger) Printf(format string, v ...interface{}) {
	l.tb.Helper()
	l.tb.Log(fmt.Sprintf(format, v...))
}

func (l *TestLogger) Println(v ...interface{}) {
	l.tb.Helper()
	l.tb.Log(fmt.Sprintln(v...))
}

func (l *TestLogger) Print(v ...interface{}) {
	l.tb.Helper()
	l.tb.Log(fmt.Sprint(v...))
}

func (l *TestLogger) Debugf(format string, v ...interface{}) {
	l.tb.Helper()
	l.tb.Log("debug: " + fmt.Sprintf(format, v...))
}

func (l *TestLogger) Debugln(v ...interface{}) {
	l.tb.Helper()
	l.tb.Log("debug: " + fmt.Sprintln(v...))
}

func (l *TestLogger) Debug(v ...interface{}) {
	l.tb.Helper()
	l.tb.Log("debug: " + fmt.Sprint(v...))
}

func (l *TestLogger) Fatalf(format string, v ...interface{}) {
	l.tb.Helper()
	l.tb.Fatalf(format, v...)
}
