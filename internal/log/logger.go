package log

import (
	"io"
	"log"
	"os"
	"sync"
	"sync/atomic"
)

type Logger interface {
	SetOutput(w io.Writer)
	Printf(format string, v ...any)
	Print(v ...any)
	Println(v ...any)
	Fatal(v ...any)
	Fatalf(format string, v ...any)
	Fatalln(v ...any)
	Panic(v ...any)
	Panicf(format string, v ...any)
	Panicln(v ...any)
	Prefix() string
	SetPrefix(prefix string)
	Writer() io.Writer
}

type Level uint32

const (
	None Level = iota << 1
	ServerLevel
	PassiveLevel
	CommandLevel
	DebugLevel
	All = ServerLevel | PassiveLevel | CommandLevel | DebugLevel
)

type logger struct {
	level atomic.Uint32
	mu    sync.Mutex
	Logger
}

var std = newLogger(log.New(os.Stderr, "", log.LstdFlags), None)

func newLogger(l Logger, level Level) *logger {
	lgr := &logger{
		Logger: l,
	}
	lgr.SetLevel(level)
	return lgr
}

func SetLogger(l Logger) {
	std.mu.Lock()
	defer std.mu.Unlock()
	std.Logger = l
}

func SetOutput(w io.Writer) {
	std.SetOutput(w)
}

func SetLevel(level Level) {
	std.SetLevel(level)
}

func IsEnabled(level Level) bool {
	return std.level.Load()&uint32(level) != 0
}

func (l *logger) SetLevel(level Level) {
	l.level.Store(uint32(level))
}

func (l *logger) print(prefix string, v ...any) {
	l.Print(append([]any{prefix}, v...)...)
}

func (l *logger) printf(prefix string, format string, v ...interface{}) {
	l.Printf(prefix+format, v...)
}

func Debug(v ...any) {
	if IsEnabled(DebugLevel) {
		std.print("[***] ", v...)
	}
}

func Debugf(format string, v ...any) {
	if IsEnabled(DebugLevel) {
		std.printf("[***] ", format, v...)
	}
}

func Command(v ...any) {
	if IsEnabled(CommandLevel) {
		std.print("[cmd] ", v...)
	}
}

func Commandf(format string, v ...any) {
	if IsEnabled(CommandLevel) {
		std.printf("[cmd] ", format, v...)
	}
}

func Passive(v ...any) {
	if IsEnabled(PassiveLevel) {
		std.print("[psv] ", v...)
	}
}

func Passivef(format string, v ...any) {
	if IsEnabled(PassiveLevel) {
		std.printf("[psv] ", format, v...)
	}
}

func Server(v ...any) {
	if IsEnabled(ServerLevel) {
		std.print("[res] ", v...)
	}
}

func Serverf(format string, v ...any) {
	if IsEnabled(ServerLevel) {
		std.printf("[res] ", format, v...)
	}
}

func Error(v ...any) {
	std.print("[ERRO] ", v...)
}

func Errorf(format string, v ...any) {
	std.printf("[ERRO]", format, v...)
}

func Warning(v ...any) {
	std.print("[WARN] ", v...)
}

func Warningf(format string, v ...any) {
	std.printf("[WARN]", format, v...)
}

func Fatal(v ...any) {
	std.Fatal(v...)
}

func Fatalf(format string, v ...any) {
	std.Fatalf(format, v...)
}

func Panic(v ...any) {
	std.Panic(v...)
}

func Panicf(format string, v ...any) {
	std.Panicf(format, v...)
}
