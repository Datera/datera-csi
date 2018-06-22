package main

import (
	"flag"
	"os"

	"github.com/Datera/datera-csi/driver"
	log "github.com/sirupsen/logrus"
)

const (
	usageTemplate = `INSERT TEMPLATE TEXT`
)

var (
	endpoint = flag.String("endpoint", "unix:///var/lib/kubelet/plugins/io.datera.csi.debs/csi.sock", "CSI endpoint")
	username = flag.String("username", "", "Datera Account Username")
	password = flag.String("password", "", "Datera Account Password")
	url      = flag.String("url", "", "Datera API URL (including port)")
)

func Usage() {
	log.Fatal("You used it wrong dummy")
}

func Main() int {
	flag.Parse()

	if *username == "" || *password == "" || *url == "" {
		Usage()
	}
	driver, err := driver.NewDateraDriver(*endpoint, *username, *password, *url)
	if err != nil {
		log.Fatal(err)
	}

	if err := driver.Run(); err != nil {
		log.Fatal(err)
	}
	return 0
}

func init() {

}

func main() {
	os.Exit(Main())
}
