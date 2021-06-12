package jump

import (
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/didip/tollbooth"
	"github.com/eyedeekay/sam3"
	"github.com/eyedeekay/sam3/helper"
	"github.com/eyedeekay/sam3/i2pkeys"
	"github.com/justinas/nosurf"
)

var network_template string = `<html>
<head>
</head>
<body>
  <style>
  body {
    font-family: monospace;
    font-size: large;
  }
  b    {
    color: black;
  }
  p    {
    color: darkgrey;
  }
  input, textarea {
    position: sticky;
    left: 20%;
    width: 70%;
  }
  </style>
  <h1>Jump-Transparency Host: {{ .Me.Name }} </h1>
  <div>This is an instance of Jump-Transparency, an I2P Site which specializes in registering
  and distributing human-readble hostnames across the I2P network. It is designed to
  be simple to run and extremely stable, but also has some unique features that make
  it a little different than other jump hosts.</div></br>
  <div>For one thing, Jump-Transparency is nearly zero-configuration to get running. There is no
  complicated setting up tunnels. If you want to run a jump service, just running the
  executable will result in a working deployment. It uses SAM to set up it's own tunnels
  without the intervention of the user. This makes it extremely easy to host your own.</div></br>
  <div>Besides that, Jump-Transparency has features for sharing addresses from other jump services.
  It can collate multiple hosts.txt files from many sources, and distribute them as a
  secondary <code>hosts.txt</code> file called "<code>peer-hosts.txt<code>," enabling
  networks of jump operators to provide eachother with redundancy. Jump-Transparency servers also
  accept a single "Announce" daily from other services that want to publicize themselves.
  </div></br>

  <div>
    <h2>Trust Chart</h2>
    <div>The trust chart is an experimental feature. It takes all of the peers
    and identifies whether they have the same hostnames in their address book.
    It then compares the hostnames by base32 address, and determines whether they
    agree with the hosts.txt file we use. It also displays what each service believes
    the Base64 Destination is. Sometimes these differ due to the implementation
    differences in name registries, even though they are actually the same
    destination.
    </div>
    <ul>
      <li><a href="http://{{ .I2PAddr.Base32 }}/trust"><b>Visit the Trust Chart:</b></a></li>
    </ul>
  </div>

  <div>
    <h2>Subscription URL's</h2>
    <ul>
      <li><b>Hosts File Subscription:</b> http://{{ .I2PAddr.Base32 }}/hosts.txt
        <ul>
          <li>This hosts.txt file contains only hosts which were registered at this
          service.</li>
        </ul>
      </li>
      <li><b>Peer Hosts Files Subscription:</b> http://{{ .I2PAddr.Base32 }}/peer-hosts.txt
        <ul>
          <li>This hosts.txt file contains the combined hosts files of many peers,
          All the addresses within are from other jump services listed below.</li>
        </ul>
      </li>
    </ul>
    <h3>Jump-Transparency Peers</h3>
    <div>
    </div>
    <div>
      {{range $index, $element := .Peers}} {{$index}} {{$element.Name}} <a href="{{$element.MyURL}}">{{$element.MyURL}}</a> <a href="peer-{{$element.Name}}-hosts.txt"> Mirror hosts.txt </a> </br>  {{else}} This server is not configured to mirror any peer addresses. {{end}}
    </div>
    <h3>Announces</h3>
    <div>
      Announces are triggered remotely and are strictly rate-limited to one per client per day.
      Announces expiere after 24 hours and must be renewed daily. This makes them an alternative
      way of announcing your site's up-time. It is up to the discretion of the announcer to decide
      what type of address to advertise. It may be a b32, a hostname, or an addresshelper link.
      To announce a site, send a <code>POST<code> request to http://{{ .I2PAddr.Base32 }}/announce
      with the body: <pre><code>
      host_name=$NAME_OF_YOUR_SITE
      host_host=$PROTOCOL_SCHEME$ADDRESS_OF_YOUR_CHOICE
      </code></pre>
    </div>
    <div>
    {{range $index, $element := .Pals}} <span>{{$element}}</span> <a href="{{$index}}">{{$index}}</a> {{else}} No one has announced a peer address yet. {{end}}
    </div>
  </div>
`

var server_template string = `
  <div>
    <h2>Hostname Registration</h2>
    <div>It is possible to register a human-readble name for your I2P site here. Simply fill
    out the small form below. The administrator of a Jump-Transparency Service always has the right to
    decline to register your human-readable name at their service. I've gone to the trouble
    of making it easy to host a jump service, so if you have a problem with it, go host your
    own.</div></br>
    <div>There is a firm limit of one and only one hostname request per client per day.</div></br>
    <form action="/hostadd" method="post">
      <label for="hostname">Preferred Hostname:</label>
      <input type="text" id="hostname" name="host_name"></br>
      <label for="destination">Authentication String:</label>
      <input type="text" id="destination" name="host_destination"></br>
      <label for="description">Short Description:</label>
      <textarea id="description" name="host_description"></textarea></br>
      <button type="submit">Submit your hostname</button>
    </form>
  </div>
`

var ops_template string = `
</body>
`

var default_template string = network_template + server_template + ops_template

type WebServer struct {
	Me        *I2PJump
	Queue     *I2PJump
	Peers     []*I2PJump
	Pals      map[string]string
	Templates map[string]string
	KeysPath  string
	Homepage  string
	samaddr   string
	I2PAddr   *i2pkeys.I2PAddr
}

func (ws *WebServer) Base32() string {
	return ws.I2PAddr.Base32()
}

func (ws *WebServer) AgglomeratedHostsFile() []byte {
	var returnable []byte
	var unreturnable []byte
	for _, v := range ws.Peers {
		unreturnable = append(unreturnable, v.HostsFile()...)
		returnable = unreturnable //[]byte(strings.Replace(string(unreturnable), "\n", "", -1))
	}
	return returnable
}

func (ws *WebServer) TrustCheck(hostname string) (agrees map[string]bool, votes map[string]string, host string) {
	myval, ok := ws.Me.ToMap()[hostname]
	host = hostname
	agrees = make(map[string]bool)
	votes = make(map[string]string)
	if ok {
		for _, peer := range ws.Peers {
			val, ok := peer.ToMap()[hostname]
			if ok {
				myaddr, err := i2pkeys.NewI2PAddrFromString(myval)
				if err == nil {
					valaddr, err := i2pkeys.NewI2PAddrFromString(myval)
					if err == nil {
						if valaddr.Base32() == myaddr.Base32() {
							agrees[peer.Name] = true
						} else {
							agrees[peer.Name] = false
						}
						votes[peer.Name] = val
					}
				}
			}
		}
	}
	return
}

func (ws *WebServer) TrustCheckElement(agrees map[string]bool, votes map[string]string, hostname string) string {
	var r string
	if len(agrees) > 0 && len(votes) > 0 {
		r += `<div class="server_` + hostname + `">`
		r += `  <h1 class="server_` + hostname + `"> Hostname ` + hostname + `</h2>`
		for peerindex, agree := range agrees {
			r += `<div class="server_` + peerindex + `">`
			r += `  <h2 class="server_` + peerindex + `"> Server ` + peerindex + `</h2>`

			if agree {
				r += `  <h4 class="server_` + peerindex + `">`
				r += `    Agrees with us about the base32 destination`
				r += `  </h4>`
			} else {
				r += `  <h4 class="server_` + peerindex + `">`
				r += `    Disagrees with us about the base32 destination`
				r += `  </h4>`
			}
			r += `  <div class="server_` + peerindex + `">`
			r += `    Sees the base64 address as: ` + votes[peerindex]
			r += `  </div>`
			r += `</div>`
		}
		r += `</div>`
	}
	return r
}

func (ws *WebServer) TrustChart() string {
	err := ioutil.WriteFile("all-known-hosts.txt", []byte(ws.AgglomeratedHostsFile()), 0644)
	if err != nil {
		return "Error rendering page, please contact the admin"
	}
	virtjump, err := NewI2PJump("all-known-hosts.txt", ws.samaddr, ws.I2PAddr.Base32(), "http://"+ws.I2PAddr.Base32()+"/peer-hosts.txt")
	if err != nil {
		return "Error rendering page, please contact the admin"
	}
	if virtjump != nil {
		var ret string
		ret += `<html>
<head>
</head>
<body>
  <style>
  body {
    font-family: monospace;
    font-size: large;
  }
  b    {
    color: black;
  }
  p    {
    color: darkgrey;
  }
  input, textarea {
    position: sticky;
    left: 20%;
    width: 70%;
  }
  </style>`
		for host := range virtjump.ToMap() {
			log.Println("Checking host:", host)
			ret += ws.TrustCheckElement(ws.TrustCheck(host))
		}
		ret += `</body>
</html>`
		return ret
	}
	return "Trustchart closed"
}

func (ws WebServer) CheckLoop() error {
	time.Sleep(time.Minute * 5)
	log.Println("Initiating re-check cycles")
	for {
		ws.Recheck(60)
	}
	return nil
}
func (ws WebServer) Recheck(delay int) error {
	for _, peer := range ws.Peers {
		e := peer.Fetch()
		if e != nil {
			log.Printf("Error fetching peer hosts.txt: %s %s", peer.Name, e.Error())
			//					return nil, e
		}
		ws.Peers = append(ws.Peers, peer)
		time.Sleep(time.Minute * time.Duration(delay))
	}
	return nil
}

func (ws WebServer) ValidateHostAnnounce(hosthost string) error {
	session, err := sam.I2PStreamSession("eph", ws.samaddr, "sam-"+"validator-client")
	if err != nil {
		return err
	}
	log.Printf("looking up: %s", hosthost)
	hostname, err := session.Lookup(hosthost)
	if err != nil {
		return err
	}
	log.Println("validated host", hostname)
	session.Close()
	return nil
}

func (ws WebServer) ServeHTTP(rw http.ResponseWriter, rq *http.Request) {
	switch rq.URL.Path {
	case "/recheck":
		err := ws.Recheck(1)
		if err != nil {
			log.Println("Force recheck error", err)
		}
		rw.Write([]byte("Forcing recheck of all peers"))
	case "/trust":
		rw.Write([]byte(ws.TrustChart()))
	case "/hosts.txt":
		rw.Write(ws.Me.HostsFile())
	case "/peer-hosts.txt":
		rw.Write(ws.AgglomeratedHostsFile())
	case "/announce":
		hostname := rq.FormValue("host_name")
		base32 := rq.FormValue("host_host")
		if ws.ValidateHostAnnounce(base32) == nil {
			ws.Pals[base32] = hostname
		}
	default:
		if strings.HasPrefix(rq.URL.Path, "/peer-") {
			if strings.HasSuffix(rq.URL.Path, "-hosts.txt") {
				str := strings.TrimRight(strings.TrimLeft(rq.URL.Path, "/peer-"), "-hosts.txt")
				for _, v := range ws.Peers {
					if v.Name == str {
						rw.Write(v.HostsFile())
					}
				}
			}
		} else if strings.HasPrefix(rq.URL.Path, "/jump.cgi") {
			addrpair := strings.SplitN(rq.URL.Path, `?a=`, 2)
			if len(addrpair) == 2 {
				domain := strings.SplitN(addrpair[1], `.i2p`, 2)[0]
				if entry, ok := ws.Me.HostsTxt.ToMap()[domain]; ok {
					http.Redirect(rw, rq, "http://"+domain+"/?i2padddresshelper"+entry, 301)
				}
			}
		} else if strings.HasPrefix(rq.URL.Path, "/cgi-bin/jump.cgi") {
			addrpair := strings.SplitN(rq.URL.Path, `?a=`, 2)
			if len(addrpair) == 2 {
				domain := strings.SplitN(addrpair[1], `.i2p`, 2)[0]
				if entry, ok := ws.Me.HostsTxt.ToMap()[domain]; ok {
					http.Redirect(rw, rq, "http://"+domain+"/?i2padddresshelper"+entry, 301)
				}
			}
		} else if strings.HasPrefix(rq.URL.Path, "/jump") {
			addrpair := strings.SplitN(rq.URL.Path, `?a=`, 2)
			if len(addrpair) == 2 {
				domain := strings.SplitN(addrpair[1], `.i2p`, 2)[0]
				if entry, ok := ws.Me.HostsTxt.ToMap()[domain]; ok {
					http.Redirect(rw, rq, "http://"+domain+"/?i2padddresshelper"+entry, 301)
				}
			}
		} else if strings.HasPrefix(rq.URL.Path, "/hostadd") {
			hostname := rq.FormValue("host_name")
			destination := rq.FormValue("host_destination")
			description := rq.FormValue("host_description")
			log.Printf("client attempted to register: %s %s %s", hostname, destination, description)
		} else {
			rw.Header().Add("Content-Type", "text/html")
			tmp := strings.Split(rq.URL.Path, "/")
			lang := "en"
			if len(tmp) > 1 {
				cleaned := strings.Replace(tmp[1], "/", "", -1)
				if cleaned == "" {
					lang = "en"
				} else {
					lang = cleaned
				}
			}
			log.Printf("Rendering language: %d %s, %s", len(tmp), lang, ws.Templates[lang])
			tmpl, err := template.New(ws.Me.Name).Parse(ws.Templates[lang])
			if err != nil {
				log.Printf("Template generation error, %s", err)
			}
			err = tmpl.Execute(rw, ws)
			if err != nil {
				log.Printf("Template execution error, %s", err)
			}
		}
	}
}

func NewWebServer(name, samaddr, keyspath, hostsfile string, peerslist []string, addr *i2pkeys.I2PAddr) (*WebServer, error) {
	var ws WebServer
	var e error
	ws.I2PAddr = addr
	log.Println(ws.Base32())
	ws.Me, e = NewI2PJump(hostsfile, samaddr, name, "")
	ws.Queue, e = NewI2PJump(name+"-queue.txt", samaddr, name+"-queue", "")
	ws.Templates = make(map[string]string)
	ws.Pals = make(map[string]string)
	ws.Templates["en"] = default_template
	ws.samaddr = samaddr //"127.0.0.1:7656"
	if e != nil {
		return nil, e
	}

	for i, v := range peerslist {
		V := strings.SplitN(v, "=", 2)
		if len(V) == 2 {
			peer, e := NewI2PJump(V[0]+".txt", samaddr, V[0], V[1])
			if e != nil {
				return nil, e
			}
			secs := (i * 5)
			log.Println("Sleeping", secs, "seconds")
			go func() {
				time.Sleep(time.Second * time.Duration(secs))
				e = peer.Fetch()
				if e != nil {
					log.Printf("Error fetching peer hosts.txt: %s %s", peer.Name, e.Error())
					//					return nil, e
				}
				ws.Peers = append(ws.Peers, peer)
			}()
		}

	}
	go ws.CheckLoop()
	return &ws, nil
}

type I2PServer struct {
	*sam3.StreamListener
	*WebServer
	Name      string
	SAMAddr   string
	KeysPath  string
	HostsFile string
}

func (is *I2PServer) Serve() error {
	limiter := tollbooth.NewLimiter(1, nil)
	//1,
	limiter.SetOnLimitReached(func(w http.ResponseWriter, r *http.Request) {
		log.Println("LIMITER", r.URL.Path)
		switch r.URL.Path {
		case "/hostadd":
		case "/announce":
		default:
			log.Println("EXEMPTING LIMITER", r.URL.Path)
			//			limiter
		}
	})
	configuredHandler := nosurf.New(tollbooth.LimitHandler(limiter, is.WebServer))
	configuredHandler.ExemptPath("/announce")
	return http.Serve(is.StreamListener, configuredHandler)
}

func NewI2PServer(name, samaddr, keyspath, hostsfile string, peerslist []string) (*I2PServer, error) {
	var is I2PServer
	var e error
	if name == "" {
		name = "TODORANDOMNAME"
	}
	if samaddr == "" {
		samaddr = "127.0.0.1:7657"
	}
	if keyspath == "" {
		keyspath = "jump"
	}
	if hostsfile == "" {
		hostsfile = "hosts.txt"
	}
	is.Name = name
	is.SAMAddr = samaddr
	is.KeysPath = keyspath
	is.HostsFile = hostsfile
	if _, err := os.Stat(is.HostsFile); !os.IsNotExist(err) {
		if _, err := os.Stat(is.HostsFile + ".orig"); os.IsNotExist(err) {
			bytes, err := ioutil.ReadFile(is.HostsFile)
			if err != nil {
				return nil, err
			}
			err = ioutil.WriteFile(is.HostsFile+".orig", bytes, 0644)
			if err != nil {
				return nil, err
			}
		}
	}
	is.StreamListener, e = sam.I2PListener(is.Name, is.SAMAddr, is.KeysPath)
	if e != nil {
		return nil, e
	}
	addr := is.StreamListener.Addr().(i2pkeys.I2PAddr)
	is.WebServer, e = NewWebServer(name, samaddr, keyspath, hostsfile, peerslist, &addr)
	if e != nil {
		return nil, e
	}
	return &is, e
}
