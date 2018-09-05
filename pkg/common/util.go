package common

import (
	"context"
	"os/exec"
	"strings"
)

var (
	execCommand = exec.Command
)

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
