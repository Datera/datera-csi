package common

import (
	"context"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	ReqName = "req"
	TraceId = "tid"
)

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
