package log

import (
	"fmt"
	"io"
	"log"
	"os"
)

type Logger struct {
	l     *log.Logger
	debug bool
}

func New(out io.Writer) *Logger {
	return &Logger{
		l: log.New(out, "", log.LstdFlags),
	}
}

func NewDebugLogger(out io.Writer) *Logger {
	return &Logger{
		l:     log.New(out, "", log.Lshortfile|log.LstdFlags),
		debug: true,
	}
}

// Printf calls l.Output to print to the logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Printf(format string, v ...interface{}) {
	l.l.Output(2, fmt.Sprintf(format, v...))
}

// Print calls l.Output to print to the logger.
// Arguments are handled in the manner of fmt.Print.
func (l *Logger) Print(v ...interface{}) { l.l.Output(2, fmt.Sprint(v...)) }

// Println calls l.Output to print to the logger.
// Arguments are handled in the manner of fmt.Println.
func (l *Logger) Println(v ...interface{}) { l.l.Output(2, fmt.Sprintln(v...)) }

//Debugf calls l.Output to print to the logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Debugf(format string, v ...interface{}) {
	if !l.debug {
		return
	}
	l.l.Output(2, "debug: "+fmt.Sprintf(format, v...))
}

//Debug calls l.Output to print to the logger.
// Arguments are handled in the manner of fmt.Print.
func (l *Logger) Debug(v ...interface{}) {
	if !l.debug {
		return
	}
	l.l.Output(2, "debug: "+fmt.Sprint(v...))
}

//Debugln calls l.Output to print to the logger.
// Arguments are handled in the manner of fmt.Println.
func (l *Logger) Debugln(v ...interface{}) {
	if !l.debug {
		return
	}
	l.l.Output(2, "debug: "+fmt.Sprintln(v...))
}

// Fatalf is equivalent to l.Printf() followed by a call to os.Exit(1).
func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.l.Output(2, fmt.Sprintf(format, v...))
	os.Exit(1)
}
