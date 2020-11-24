package jump

import (
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

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
  <h1>Jump Host: {{ .Me.Name }} </h1>
  <div>This is an instance of I2PJump, an I2P Site which specializes in registering
  and distributing human-readble hostnames across the I2P network. It is designed to
  be simple to run and extremely stable, but also has some unique features that make
  it a little different than other jump hosts.</div></br>
  <div>For one thing, I2PJump is nearly zero-configuration to get running. There is no
  complicated setting up tunnels. If you want to run a jump service, just running the
  executable will result in a working deployment. It uses SAM to set up it's own tunnels
  without the intervention of the user. This makes it extremely easy to host your own.</div></br>
  <div>Besides that, I2PJump has features for sharing addresses from other jump services.
  It can collate multiple hosts.txt files from many sources, and distribute them as a
  secondary <code>hosts.txt</code> file called "<code>peer-hosts.txt<code>," enabling
  networks of jump operators to provide eachother with redundancy. I2PJump servers also
  accept a single "Announce" daily from other services that want to publicize themselves.
  </div></br>

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
    <h3>Jump Peers</h3>
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
      host_host=$ADDRESS_OF_YOUR_CHOICE
      </code></pre>
    </div>
    <div>
    {{range $index, $element := .Pals}} {{$index}} {{$element.Name}} <a href="{{$element.MyURL}}">{{$element.MyURL}}</a> <a href="peer-{{$element.Name}}-hosts.txt"> Mirror hosts.txt </a> </br>  {{else}} This server is not configured to mirror any peer addresses. {{end}}
    </div>
  </div>
`

var server_template string = `
  <div>
    <h2>Hostname Registration</h2>
    <div>It is possible to register a human-readble name for your I2P site here. Simply fill
    out the small form below. The administrator of a Jump Service always has the right to
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
	Me        *Jump
	Queue     *Jump
	Peers     []*Jump
	Pals      map[string]string
	Templates map[string]string
	KeysPath  string
	Homepage  string
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
		returnable = []byte(strings.Replace(string(unreturnable), "\n", "", -1))
	}
	return returnable
}

func (ws WebServer) ServeHTTP(rw http.ResponseWriter, rq *http.Request) {
	switch rq.URL.Path {
	case "/hosts.txt":
		rw.Write(ws.Me.HostsFile())
	case "/peer-hosts.txt":
		rw.Write(ws.AgglomeratedHostsFile())
	case "/announce":
		hostname := rq.FormValue("host_name")
		base32 := rq.FormValue("host_host")
		ws.Pals[base32] = hostname
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
	ws.Me, e = NewJump(hostsfile, samaddr, name, "")
	ws.Queue, e = NewJump(name+"-queue.txt", samaddr, name+"-queue", "")
	ws.Templates = make(map[string]string)
	ws.Pals = make(map[string]string)
	ws.Templates["en"] = default_template
	if e != nil {
		return nil, e
	}

	for _, v := range peerslist {
		V := strings.SplitN(v, "=", 2)
		if len(V) == 2 {
			peer, e := NewJump(V[0]+".txt", samaddr, V[0], V[1])
			if e != nil {
				return nil, e
			}
			e = peer.Fetch()
			if e != nil {
			  log.Printf("Error fetching peer hosts.txt: %s %s", peer.Name, e.Error())
				return nil, e
			}
			ws.Peers = append(ws.Peers, peer)
		}
	}
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
	limiter := tollbooth.NewLimiter(.0000115, nil)
	limiter.SetOnLimitReached(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/hostadd" {
			if r.URL.Path != "/announce" {
				is.WebServer.ServeHTTP(w, r)
			}
		}
	})
	return http.Serve(is.StreamListener, nosurf.New(tollbooth.LimitFuncHandler(limiter, is.WebServer.ServeHTTP)))
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
