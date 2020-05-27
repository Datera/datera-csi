package common

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	uuid "github.com/google/uuid"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"

	dsdk "github.com/Datera/go-sdk/pkg/dsdk"
	csi "github.com/container-storage-interface/spec/lib/go/csi"
)

const (
	Ext4 = "ext4"
	Xfs  = "xfs"
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

func WithCtxt(ctxt context.Context, reqName, traceId string) context.Context {
	if traceId == "" {
		traceId = GenId()
	}
	ctxt = context.WithValue(topctxt, TraceId, traceId)
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

func GetCaptureGroups(r *regexp.Regexp, matchString string) map[string]string {
	match := r.FindStringSubmatch(matchString)
	result := make(map[string]string)
	if len(match) != 0 {
		for i, name := range r.SubexpNames() {
			if i != 0 && name != "" {
				result[name] = match[i]
			}
		}
	}
	return result
}

func DatVersionGte(v1, v2 string) (bool, error) {
	a, err := versionToInt(v1)
	if err != nil {
		return false, err
	}
	b, err := versionToInt(v2)
	if err != nil {
		return false, err
	}
	return a >= b, nil
}

func versionToInt(v string) (int, error) {
	// Using a factor of 100 per digit so up to 100 versions are supported
	// per major/minor/patch/subpatch digit in this calculation
	VersionDigits := 4
	factor := math.Pow10(VersionDigits * 2)
	div := math.Pow10(2)
	result := 0
	for _, c := range strings.Split(v, ".") {
		i, err := strconv.ParseInt(c, 10, 0)
		if err != nil {
			return 0, err
		}
		result += int(i) * int(factor)
		factor /= div
	}
	return result, nil
}

func StripSecretsAndGetChapParams(req interface{}) (map[string]string) {

	stripNeeded := false
	chapParams := map[string]string{}
	secrets := map[string]string{}

	switch reqType := req.(type) {
		case *csi.CreateVolumeRequest:
			secrets = reqType.Secrets
			stripNeeded = true
		case *csi.NodeStageVolumeRequest:
			secrets = reqType.Secrets
			stripNeeded = true
		case *csi.DeleteVolumeRequest:
			secrets = reqType.Secrets
			stripNeeded = true
		default:
			return secrets
	}

	if stripNeeded == true {

		if value, exists := secrets["node.session.auth.username"]; exists {
			secrets["node.session.auth.username"] = "***stripped***"
			chapParams["node.session.auth.username"] = value
		}
		if value, exists := secrets["node.session.auth.password"]; exists {
			secrets["node.session.auth.password"] = "***stripped***"
			chapParams["node.session.auth.password"] = value
		}
		if value, exists := secrets["node.session.auth.username_in"]; exists {
			secrets["node.session.auth.username_in"] = "***stripped***"
			chapParams["node.session.auth.username_in"] = value
		}
		if value, exists := secrets["node.session.auth.password_in"]; exists {
			secrets["node.session.auth.password_in"] = "***stripped***"
			chapParams["node.session.auth.password_in"] = value
		}

	}
	return chapParams

}
