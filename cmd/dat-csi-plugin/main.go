package main

import (
	"flag"
	"os"

	driver "github.com/Datera/datera-csi/driver"
	log "github.com/sirupsen/logrus"

	udc "github.com/Datera/go-udc/pkg/udc"
)

const (
	usageTemplate = `INSERT TEMPLATE TEXT`
)

var (
	endpoint = flag.String("endpoint", "unix:///var/lib/kubelet/plugins/io.datera.csi.dsp/csi.sock", "CSI endpoint")
)

func Usage() {
	log.Fatal("You used it wrong dummy")
}

func Main() int {
	flag.Parse()

	conf, err := udc.GetConfig()
	if err != nil {
		log.Fatal(err)
	}
	log.Info("Using Universal Datera Config")
	udc.PrintConfig()
	driver, err := driver.NewDateraDriver(*endpoint, conf)
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
