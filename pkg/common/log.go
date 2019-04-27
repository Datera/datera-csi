package common

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	ReqName = "req"
	TraceId = "tid"
)

// DecorateRuntimeContext appends line, file and function context to the logger
func DecorateRuntimeContext(logger *log.Entry) *log.Entry {
	if pc, file, line, ok := runtime.Caller(3); ok {
		fName := runtime.FuncForPC(pc).Name()
		return logger.WithField("file", file).WithField("line", line).WithField("func", fName)
	} else {
		return logger
	}
}

func logHelper(ctxt context.Context) *log.Entry {
	reqname := ctxt.Value(ReqName).(string)
	tid := ctxt.Value(TraceId).(string)
	return DecorateRuntimeContext(log.WithFields(log.Fields{
		ReqName: reqname,
		TraceId: tid,
	}))
}

func Debug(ctxt context.Context, s interface{}) {
	logHelper(ctxt).Debug(s)
}

func Debugf(ctxt context.Context, s string, args ...interface{}) {
	s = checkArgs(ctxt, s, args...)
	logHelper(ctxt).Debugf(s, args...)
}

func Info(ctxt context.Context, s interface{}) {
	logHelper(ctxt).Info(s)
}

func Infof(ctxt context.Context, s string, args ...interface{}) {
	s = checkArgs(ctxt, s, args...)
	logHelper(ctxt).Infof(s, args...)
}

func Warning(ctxt context.Context, s interface{}) {
	logHelper(ctxt).Warning(s)
}

func Warningf(ctxt context.Context, s string, args ...interface{}) {
	s = checkArgs(ctxt, s, args...)
	logHelper(ctxt).Warningf(s, args...)
}

func Error(ctxt context.Context, s interface{}) {
	logHelper(ctxt).Error(s)
}

func Errorf(ctxt context.Context, s string, args ...interface{}) {
	s = checkArgs(ctxt, s, args...)
	logHelper(ctxt).Errorf(s, args...)
}

func Fatal(ctxt context.Context, s interface{}) {
	logHelper(ctxt).Fatal(s)
}

func Fatalf(ctxt context.Context, s string, args ...interface{}) {
	s = checkArgs(ctxt, s, args...)
	logHelper(ctxt).Fatalf(s, args...)
}

// Hack just to make sure I don't miss these
func checkArgs(ctxt context.Context, s string, args ...interface{}) string {
	c := 0
	for _, f := range []string{"%s", "%f", "%d", "%v", "%#v", "%t", "%p", "%+v"} {
		c += strings.Count(s, f)
	}
	l := len(args)
	if c != l {
		Warningf(ctxt, "Wrong number of args for format string, [%d != %d]\n", l, c)
	}
	if !strings.HasSuffix(s, "\n") {
		s = strings.Join([]string{s, "\n"}, "")
	}
	return s
}

type LogFormatter struct {
}

func (f *LogFormatter) Format(entry *log.Entry) ([]byte, error) {
	msg := entry.Message
	level := entry.Level
	t := entry.Time
	fstring := ""
	for f, v := range entry.Data {
		switch v.(type) {
		case int, int32, int64, uint, uint32, uint64:
			fstring = strings.Join([]string{fstring, fmt.Sprintf("%s: %d", f, v)}, " ")
		default:
			fstring = strings.Join([]string{fstring, fmt.Sprintf("%s: %s", f, v)}, " ")
		}
	}
	// if err != nil {
	// 	fmt.Printf("Error marshalling fields during logging: %s\n", err)
	// }
	return []byte(fmt.Sprintf("%s %s %s ||> %s\n",
		t.Format(time.RFC3339),
		strings.ToUpper(level.String()),
		string(strings.TrimSpace(msg)),
		fstring),
	), nil
}

func init() {
	log.SetFormatter(&LogFormatter{})
	log.SetLevel(log.DebugLevel)
}
