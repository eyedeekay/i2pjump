package jump

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/url"

	"github.com/eyedeekay/sam3/helper"
)

type Jump struct {
	*HostsTxt
	SAMAddr string
	Name    string
	MyURL   *url.URL
}

func NewJump(hostFile, samAddr, name, jumpUrl string) (*Jump, error) {
	var j Jump
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

func (j *Jump) Fetch() error {
	session, err := sam.I2PStreamSession(j.Name, j.SAMAddr, "sam-"+j.Name+"-client")
	if err != nil {
		return err
	}
	log.Printf("DIALING: %s", j.MyURL.Host)
	conn, err := session.Dial("i2p", j.MyURL.Host+":80")
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
	log.Printf("GOT: %s", bytes)
	err = ioutil.WriteFile("peer-"+j.Name+"-hosts.txt", bytes, 0644)
	if err != nil {
		return err
	}
	j.HostsTxt, err = NewHostsTxt("peer-" + j.Name + "-hosts.txt")
	if err != nil {
		return err
	}
	return nil
}
