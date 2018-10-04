package common

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"

	uuid "github.com/google/uuid"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"

	dsdk "github.com/Datera/go-sdk/pkg/dsdk"
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

func GetHost() string {
	return host
}

func Prettify(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", " ")
	return string(b)
}

func WithCtxt(ctxt context.Context, reqName string) context.Context {
	ctxt = context.WithValue(topctxt, TraceId, GenId())
	ctxt = context.WithValue(ctxt, ReqName, reqName)
	return ctxt
}

func RunCmd(ctxt context.Context, cmd ...string) (string, error) {
	ncmd := []string{}
	for _, c := range cmd {
		c = strings.TrimSpace(c)
		if c != "" {
			ncmd = append(ncmd, c)
		}
	}
	Debugf(ctxt, "Running command: [%s]\n", strings.Join(ncmd, " "))
	prefix := ncmd[0]
	ncmd = ncmd[1:]
	c := execCommand(prefix, ncmd...)
	out, err := c.CombinedOutput()
	sout := string(out)
	Debug(ctxt, sout)
	return sout, err
}

func GenName(name string) string {
	if name == "" {
		name = GenId()
	}
	// Truncate display name to 58 characters
	maxL := 58
	if len(name) > maxL {
		rns := []rune(name)
		name = string(rns[:maxL])
	}
	return strings.Join([]string{"CSI", name}, "-")
}

func GenId() string {
	return uuid.Must(uuid.NewRandom()).String()
}

func MkSnapId(vol, snap string) string {
	return strings.Join([]string{vol, snap}, ":")
}

func ParseSnapId(snapId string) (string, string) {
	parts := strings.Split(snapId, ":")
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

func GetCode(err error) codes.Code {
	return status.Code(err)
}

func IsGrpcErr(err error) bool {
	return status.Code(err) != codes.Unknown
}

func ErrTranslator(apierr *dsdk.ApiErrorResponse) error {
	if apierr.Name == "AuthFailedError" {
		return status.Errorf(codes.Unauthenticated, "%s: %s", apierr.Name, apierr.Message)
	}
	if apierr.Name == "NotFound" {
		return status.Errorf(codes.NotFound, "%s: %s", apierr.Name, apierr.Message)
	}
	return status.Errorf(codes.Unknown, "%s: %s", apierr.Name, apierr.Message)
}
