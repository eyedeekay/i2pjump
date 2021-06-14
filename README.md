jump-transparency
=================

Jump Transparency is an I2P Name Registrar/Subscription Provider with
the ability to maintain a list of "Peer" subscription providers, and compare
them across all the available sources. This is intended to provide a way to
easily spot disagreements between subscription providers and potentially,
notice compromises early, before they happen. It is a **WORK IN PROGRESS** and
probably won't increase your security in any specific way, yet.

It was written in part because of this rather depressing and slightly bizarre
paper:

 - **[Interconnection Between Darknets](https://arxiv.org/pdf/2012.05003)**

Where the paper authors suggest targeting a specific, non-criminal individual
jump service operator in a specific way. By name. Which seems bizarrely
un-academic but hey what the hell do I know.

Features
--------

 - Basic Hostname Registration
 - Subscription file generation
 - Subscription file mirroring
 - Trust-By-Agreement system for measuring domain name replication across
   services
 - Daily announcement of Base32 address helpers
 - Automatic configuration via SAM

Usage
-----

```bash
Usage of ./jump-transparency:
  -announce string
    	Comma-separated list of other Jump-Transparency jump services to "announce" ourselves to for publicity purposes.
  -hostsfile string
    	Where to store the hosts file (default "hosts.txt")
  -keyspath string
    	Where to store the long-term keys for your hidden service (default "keys")
  -name string
    	Name to use for your Jump-Transparency server (default "jumphelp")
  -peers string
    	Comma-separated list of the other I2P jump services in the form "peerone=http://peerone.i2p/hosts.txt,peertwo=http://peerone.i2p/hosts.txt" (default "root=http://i2p-projekt.i2p/hosts.txt,identiguy=http://identiguy.i2p/hosts.txt,notbob=http://nytzrhrjjfsutowojvxi7hphesskpqqr65wpistz6wa7cpajhp7a.b32.i2p//hosts.txt,inr=http://inr.i2p/alive-hosts.txt,isitup=http://isitup.i2p/hosts.txt,reg=http://reg.i2p/hosts.txt")
  -samaddr string
    	SAM address to connect to (default "127.0.0.1:7656")
  -serve
    	Download and serve the hosts you collected (default true)
```