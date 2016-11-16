package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/mxk/go-imap/imap"
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [Options] Host\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	var err error
	flag.Usage = usage
	flag.Parse()
	if len(flag.Args()) < 1 {
		flag.Usage()
		os.Exit(1)
	}
	url, err := url.Parse(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}
	if url.Scheme != "imap" {
		log.Fatal("Invalid Scheme: ", url.Scheme)
	}

	_, err = imap.DialTLS(url.Host, nil)
	if err != nil {
		log.Fatal(err)
	}
}
