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
	// read username and/or password from STDIN if not given in the URL
	if username == "" {
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Print("Username: ")
		if scanner.Scan() {
			username = scanner.Text()
		} else {
			check(scanner.Err())
		}
	}
	if password == "" {
		fmt.Print("Password: ")
		pass_bytes, err := gopass.GetPasswd()
		check(err)
		password = string(pass_bytes)
	}
	_, err = client.Login(username, password)
	check(err)
	_, err = client.Select(url.Path[1:], true)
	check(err)

	// get the envelopes of all messages
	seqSet, err := imap.NewSeqSet("1:*")
	check(err)
	cmd, err := imap.Wait(client.UIDFetch(seqSet, "BODY[HEADER]"))
	check(err)
	envelopes := map[string]uint32{}
	toDelete := []uint32{}
	for _, resp := range cmd.Data {
		info := resp.MessageInfo()
		key := fmt.Sprintf("%s", info.Attrs["BODY[HEADER]"])
		if _, ok := envelopes[key]; ok {
			// we already have this message
			toDelete = append(toDelete, info.UID)
		} else {
			envelopes[key] = info.UID
		}
	}
	fmt.Println("Number of emails:", len(cmd.Data))
	fmt.Println("Emails kept:", len(envelopes))
	fmt.Println("Emails to delete:", len(toDelete))
}
