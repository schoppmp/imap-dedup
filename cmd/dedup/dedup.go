package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/howeyc/gopass"
	"github.com/mxk/go-imap/imap"
)

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [Options] Host\nOptions:\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	var err error
	var starttls bool
	flag.Usage = usage
	flag.BoolVar(&starttls, "starttls", false, "Use STARTTLS instead of SSL/TLS")

	flag.Parse()
	if len(flag.Args()) < 1 {
		flag.Usage()
		os.Exit(1)
	}
	url, err := url.Parse("imap://" + flag.Arg(0))
	check(err)

	// establish a secure connection with the server
	var client *imap.Client
	if starttls {
		client, err = imap.Dial(url.Host)
	} else {
		client, err = imap.DialTLS(url.Host, nil)
	}
	check(err)
	defer func() { // gracefully shutdown
		_, err = client.Logout(1 * time.Second)
		check(err)
	}()
	if starttls {
		_, err := imap.Wait(client.StartTLS(nil))
		check(err)
	}

	// use username + password authentication
	var username, password string
	if url.User != nil {
		username = url.User.Username()
		password, _ = url.User.Password()
	}
	if username == "" {
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Printf("Username: ")
		if scanner.Scan() {
			username = scanner.Text()
		} else {
			check(scanner.Err())
		}
	}
	if password == "" {
		fmt.Printf("Password: ")
		pass_bytes, err := gopass.GetPasswd()
		check(err)
		password = string(pass_bytes)
	}
	_, err = client.Login(username, password)
	check(err)
	cmd, err := client.Select(url.Path[1:], true)
	check(err)
	for _, resp := range cmd.Data {
		println(resp.String())
	}
}
