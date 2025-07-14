package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	var showVersion bool
	flag.BoolVar(&showVersion, "v", false, "show version information")
	var configFile string
	flag.StringVar(&configFile, "c", "config.json", "config file")
	flag.Parse()

	if showVersion {
		fmt.Println(GetVersion())
		os.Exit(0)
	}

	err := LoadConfig(configFile)
	if err != nil {
		panic(err)
	}

	brs := NewSequencer[*BlockReceipt](0)
	bes := NewSequencer[*BlockEvent](0)

}
