package main

import (
	"flag"
	"fmt"
	"os"

	driver "github.com/Datera/datera-csi/pkg/driver"
	log "github.com/sirupsen/logrus"

	udc "github.com/Datera/go-udc/pkg/udc"
)

var (
	version = flag.Bool("version", false, "Show version information")
)

func Main() int {
	flag.Parse()

	if *version {
		fmt.Printf("Datera CSI Plugin Version: %s-%s\n", driver.Version, driver.Githash)
		os.Exit(0)
	}
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
