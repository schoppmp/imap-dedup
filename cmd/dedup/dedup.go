package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/mxk/go-imap/imap"
	"golang.org/x/crypto/ssh/terminal"
)

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [options] [user[:password]@]hostname[:port]/folder\nOptions:\n", os.Args[0])
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
	scanner := bufio.NewScanner(os.Stdin)
	if username == "" {
		fmt.Print("Username: ")
		if scanner.Scan() {
			username = scanner.Text()
		}
		check(scanner.Err())
	}
	if password == "" {
		fmt.Print("Password: ")
		if terminal.IsTerminal(int(os.Stdin.Fd())) { // read password without echo
			pass_bytes, err := terminal.ReadPassword(int(os.Stdin.Fd()))
			check(err)
			password = string(pass_bytes)
			fmt.Println()
		} else {
			if scanner.Scan() {
				password = scanner.Text()
			}
			check(scanner.Err())
		}
	}
	_, err = client.Login(username, password)
	check(err)
	_, err = client.Select(url.Path[1:], false)
	check(err)

	// get the envelopes of all messages
	seqSet, err := imap.NewSeqSet("1:*")
	check(err)
	cmd, err := imap.Wait(client.Fetch(seqSet, "BODY.PEEK[HEADER]"))
	check(err)
	envelopes := map[string]uint32{}
	toDelete := []uint32{}
	for _, resp := range cmd.Data {
		info := resp.MessageInfo()
		key := fmt.Sprintf("%s", info.Attrs["BODY[HEADER]"])
		if _, ok := envelopes[key]; ok {
			// we already have this message
			toDelete = append(toDelete, info.Seq)
		} else {
			envelopes[key] = info.Seq
		}
	}
	fmt.Println("Number of emails:", len(cmd.Data))
	fmt.Println("Number of duplicates:", len(toDelete))

	if len(toDelete) == 0 {
		fmt.Println("No messages to delete")
		os.Exit(2)
	}
	// ask for confirmation
	s := ""
	if len(toDelete) != 1 {
		s = "s"
	}
	fmt.Printf("This will delete %d message%s in %s. Do you wish to continue? (y/N)",
		len(toDelete), s, url.Path[1:])
	answer := ""
	if scanner.Scan() {
		answer = scanner.Text()
	}
	check(scanner.Err())
	if !(answer == "y" || answer == "Y") {
		fmt.Println("Aborted")
		os.Exit(2)
	}

	// mark all messages in toDelete as deleted
	seqSet = &imap.SeqSet{}
	seqSet.AddNum(toDelete...)
	cmd, err = imap.Wait(client.Store(seqSet, "+FLAGS", `\DELETED`))
	fmt.Printf("%v\n", seqSet)
	check(err)
	// close mailbox and expunge
	_, err = client.Close(true)
	fmt.Println("Messages successfully deleted")
}
