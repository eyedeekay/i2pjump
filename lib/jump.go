package jump

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"strings"

	"github.com/eyedeekay/sam3/helper"
)

type I2PJump struct {
	*HostsTxt
	SAMAddr string
	Name    string
	MyURL   *url.URL
}

func NewI2PJump(hostFile, samAddr, name, jumpUrl string) (*I2PJump, error) {
	var j I2PJump
	var e error
	j.SAMAddr = samAddr
	j.Name = name
	j.HostsTxt, e = NewHostsTxt(hostFile)
	if e != nil {
		return nil, e
	}
	j.MyURL, e = url.Parse(jumpUrl)
	if e != nil {
		return nil, e
	}
	return &j, nil
}

var literal = `
`

func (j *I2PJump) Fetch() error {
	session, err := sam.I2PStreamSession(j.Name, j.SAMAddr, "sam-"+j.Name+"-client")
	if err != nil {
		return err
	}
	log.Printf("looking up: %s", j.MyURL.Host)
	hostname, err := session.Lookup(j.MyURL.Host)
	if err != nil {
		return err
	}
	log.Printf("DIALING: %s", hostname.Base32())
	conn, err := session.DialI2P(hostname)
	if err != nil {
		return err
	}
	defer conn.Close()
	log.Printf("GETTING: %s", j.MyURL.String())
	fmt.Fprintf(conn, "GET "+j.MyURL.Path+" HTTP/1.0\r\n\r\n")
	bytes, err := ioutil.ReadAll(conn)
	if err != nil {
		return err
	}
	if strings.Contains(string(bytes), "404 Not Found") {
		return fmt.Errorf("Fetch error 404", j.Name)
	}
	log.Printf("WRITING: %s", "peer-"+j.Name+"-hosts.txt")
	err = ioutil.WriteFile("peer-"+j.Name+"-hosts.txt", bytes, 0644)
	if err != nil {
		return err
	}
	log.Printf("LOADING: %s", "peer-"+j.Name+"-hosts.txt")
	j.HostsTxt, err = NewHostsTxt("peer-" + j.Name + "-hosts.txt")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("peer-"+j.Name+"-hosts.txt", j.HostsTxt.HostsFile(), 0644)
	if err != nil {
		return err
	}
	return nil
}
