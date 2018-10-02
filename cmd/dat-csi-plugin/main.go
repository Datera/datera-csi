package main

import (
	"flag"
	"os"

	driver "github.com/Datera/datera-csi/pkg/driver"
	log "github.com/sirupsen/logrus"

	udc "github.com/Datera/go-udc/pkg/udc"
)

const (
	usageTemplate = `INSERT TEMPLATE TEXT`
)

func Main() int {
	flag.Parse()

	conf, err := udc.GetConfig()
	if err != nil {
		log.Fatal(err)
	}
	log.Info("Using Universal Datera Config")
	udc.PrintConfig()
	d, err := driver.NewDateraDriver(conf)
	if err != nil {
		log.Fatal(err)
	}

	if err := d.Run(); err != nil {
		log.Fatal(err)
	}
	return 0
}

func main() {
	os.Exit(Main())
}
