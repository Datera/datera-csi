package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	co "github.com/Datera/datera-csi/pkg/common"
	pb "github.com/Datera/datera-csi/pkg/iscsi-rpc"
	"google.golang.org/grpc"
)

const (
	address  = "unix:///iscsi-socket/iscsi.sock"
	etcIscsi = "/etc/iscsi"
	iname    = "/etc/iscsi/initiatorname.iscsi"
)

var (
	// Necessary to prevent UDC arguments from showing up
	cli = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	addr = cli.String("addr", address, "Address to send on")
	args = cli.String("args", "", "Arguments including iscsiadm prefix")
)

func setupInitName(ctxt context.Context, c pb.IscsiadmClient) {
	// co.Debugf(ctxt, "Checking for file: %s\n", addr)
	if _, err := os.Stat(etcIscsi); os.IsNotExist(err) {
		co.Debugf(ctxt, "Creating directories: %s\n", addr)
		err = os.MkdirAll(etcIscsi, os.ModePerm)
		if err != nil {
			co.Fatal(ctxt, err)
		}
		resp, err := c.GetInitiatorName(ctxt, &pb.GetInitiatorNameRequest{})
		if err != nil {
			co.Fatal(ctxt, err)
		}
		name := fmt.Sprintf("InitiatorName=%s", resp.Name)
		err = ioutil.WriteFile(iname, []byte(name), 0644)
		if err != nil {
			co.Fatal(ctxt, err)
		}
	}
}

func main() {
	cli.Parse(os.Args[1:])
	// Set up a connection to the server.
	conn, err := grpc.Dial(*addr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewIscsiadmClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	ctx = co.WithCtxt(ctx, "iscsi-send")
	defer cancel()
	setupInitName(ctx, c)
	r, err := c.SendArgs(ctx, &pb.SendArgsRequest{Args: *args})
	// iscsadm exit-status 21 is No Objects Found, which just means the system
	// is clear of other logins
	if err != nil && !strings.Contains(err.Error(), "exit status 21") {
		co.Fatalf(ctx, "Could not send args: %v", err)
	}
	if r != nil {
		fmt.Println(r.Result)
	}
}
