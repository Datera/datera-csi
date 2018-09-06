package common

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	uuid "github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

const (
	ReqName = "req"
	TraceId = "tid"
)

var (
	host    = MustS(os.Hostname())
	topctxt = context.WithValue(context.Background(), "host", host)
)

func MustS(s string, err error) string {
	if err != nil {
		panic(err)
	}
	return s
}

func Prettify(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", " ")
	return string(b)
}

func MkCtxt(reqName string) context.Context {
	ctxt := context.WithValue(topctxt, TraceId, GenId())
	ctxt = context.WithValue(ctxt, ReqName, reqName)
	return ctxt
}

func Debug(ctxt context.Context, s interface{}) {
	reqname := ctxt.Value(ReqName).(string)
	tid := ctxt.Value(TraceId).(string)
	log.WithFields(log.Fields{
		ReqName: reqname,
		TraceId: tid,
	}).Debug(s)
}

func Debugf(ctxt context.Context, s string, args ...interface{}) {
	s = checkArgs(ctxt, s, args...)
	reqname := ctxt.Value(ReqName).(string)
	tid := ctxt.Value(TraceId).(string)
	log.WithFields(log.Fields{
		ReqName: reqname,
		TraceId: tid,
	}).Debugf(s, args...)
}

func Info(ctxt context.Context, s interface{}) {
	reqname := ctxt.Value(ReqName).(string)
	tid := ctxt.Value(TraceId).(string)
	log.WithFields(log.Fields{
		ReqName: reqname,
		TraceId: tid,
	}).Info(s)
}

func Infof(ctxt context.Context, s string, args ...interface{}) {
	s = checkArgs(ctxt, s, args...)
	reqname := ctxt.Value(ReqName).(string)
	tid := ctxt.Value(TraceId).(string)
	log.WithFields(log.Fields{
		ReqName: reqname,
		TraceId: tid,
	}).Infof(s, args...)
}

func Warning(ctxt context.Context, s interface{}) {
	reqname := ctxt.Value(ReqName).(string)
	tid := ctxt.Value(TraceId).(string)
	log.WithFields(log.Fields{
		ReqName: reqname,
		TraceId: tid,
	}).Warning(s)
}

func Warningf(ctxt context.Context, s string, args ...interface{}) {
	s = checkArgs(ctxt, s, args...)
	reqname := ctxt.Value(ReqName).(string)
	tid := ctxt.Value(TraceId).(string)
	log.WithFields(log.Fields{
		ReqName: reqname,
		TraceId: tid,
	}).Warningf(s, args...)
}

func Error(ctxt context.Context, s interface{}) {
	reqname := ctxt.Value(ReqName).(string)
	tid := ctxt.Value(TraceId).(string)
	log.WithFields(log.Fields{
		ReqName: reqname,
		TraceId: tid,
	}).Error(s)
}

func Errorf(ctxt context.Context, s string, args ...interface{}) {
	s = checkArgs(ctxt, s, args...)
	reqname := ctxt.Value(ReqName).(string)
	tid := ctxt.Value(TraceId).(string)
	log.WithFields(log.Fields{
		ReqName: reqname,
		TraceId: tid,
	}).Errorf(s, args...)
}

func Fatal(ctxt context.Context, s interface{}) {
	reqname := ctxt.Value(ReqName).(string)
	tid := ctxt.Value(TraceId).(string)
	log.WithFields(log.Fields{
		ReqName: reqname,
		TraceId: tid,
	}).Fatal(s)
}

func Fatalf(ctxt context.Context, s string, args ...interface{}) {
	s = checkArgs(ctxt, s, args...)
	reqname := ctxt.Value(ReqName).(string)
	tid := ctxt.Value(TraceId).(string)
	log.WithFields(log.Fields{
		ReqName: reqname,
		TraceId: tid,
	}).Fatalf(s, args...)
}

func GenName(name string) string {
	if name == "" {
		name = GenId()
	}
	return strings.Join([]string{"CSI", name}, "-")
}

func GenId() string {
	return uuid.Must(uuid.NewRandom()).String()
}

// Hack just to make sure I don't miss these
func checkArgs(ctxt context.Context, s string, args ...interface{}) string {
	c := 0
	for _, f := range []string{"%s", "%d", "%v", "%#v", "%t", "%p", "%+v"} {
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
