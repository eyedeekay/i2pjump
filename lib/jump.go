package jump

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/url"

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

func (j *I2PJump) Fetch() error {
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
	return nil
}
