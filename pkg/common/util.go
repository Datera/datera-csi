package common

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"

	dsdk "github.com/Datera/go-sdk/pkg/dsdk"
	uuid "github.com/google/uuid"
)

var (
	execCommand = exec.Command
	host        = MustS(os.Hostname())
	topctxt     = context.WithValue(context.Background(), "host", host)
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

func RunCmd(ctxt context.Context, cmd ...string) (string, error) {
	Debugf(ctxt, "Running command: [%s]\n", strings.Join(cmd, " "))
	prefix := cmd[0]
	cmd = cmd[1:]
	c := execCommand(prefix, cmd...)
	out, err := c.CombinedOutput()
	sout := string(out)
	Debug(ctxt, sout)
	return sout, err
}

func GenName(name string) string {
	if name == "" {
		name = GenId()
	}
	// Truncate display name to 30 characters
	if len(name) > 30 {
		rns := []rune(name)
		name = string(rns[:30])
	}
	rlen := 63 - (30 + 5)
	return strings.Join([]string{"CSI", name, dsdk.RandString(rlen)}, "-")
}

func GenId() string {
	return uuid.Must(uuid.NewRandom()).String()
}
