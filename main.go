package main

import (
	"flag"
	"log"
	"strings"

	//	"i2pgit.org/idk/jump-transparency/lib"
	"github.com/eyedeekay/i2pjump/lib"
)

var (
	name      = flag.String("name", "jumphelp", "Name to use for your Jump-Transparency server")
	serve     = flag.Bool("serve", true, "Download and serve the hosts you collected")
	samaddr   = flag.String("samaddr", "127.0.0.1:7656", "SAM address to connect to")
	keyspath  = flag.String("keyspath", "keys", "Where to store the long-term keys for your hidden service")
	hostsfile = flag.String("hostsfile", "hosts.txt", "Where to store the hosts file")
	peers     = flag.String("peers", "root=http://i2p-projekt.i2p/hosts.txt,identiguy=http://identiguy.i2p/hosts.txt,notbob=http://notbob.i2p/hosts.txt,inr=http://inr.i2p/hosts.txt,isitup=http://isitup.i2p/hosts.txt,reg=http://reg.i2p/hosts.txt", "Comma-separated list of the other I2P jump services in the form \"peerone=http://peerone.i2p/hosts.txt,peertwo=http://peerone.i2p/hosts.txt\"")
	announce  = flag.String("announce", "", "Comma-separated list of other Jump-Transparency jump services to \"announce\" ourselves to for publicity purposes.")
)

func main() {
	flag.Parse()
	peerslist := strings.Split(*peers, ",")
	j, e := jump.NewI2PServer(*name, *samaddr, *keyspath, *hostsfile, peerslist)
	if e != nil {
		log.Fatal(e)
	}
	if *serve {
		if e = j.Serve(); e != nil {
			log.Fatal(e)
		}
	}
}
