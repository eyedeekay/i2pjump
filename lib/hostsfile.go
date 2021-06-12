package jump

import (
	"io/ioutil"
	"os"
	"strings"
)

type Host struct {
	Host        string
	Destination string
}

func (h *Host) String() string {
	return h.Host + "=" + h.Destination + "\n"
}

type HostsTxt struct {
	HostList []Host
	hostMap  map[string]string
}

func (ht *HostsTxt) ToMap() map[string]string {
	if len(ht.hostMap) != len(ht.HostList) {
		for _, v := range ht.HostList {
			ht.hostMap[v.Host] = v.Destination
		}
	}
	return ht.hostMap
}

func ReadHostsFile(file string) ([]string, error) {
	if file == "" {
		return []string{}, nil
	}
	bytes, err := ioutil.ReadFile(file)
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	return strings.Split(string(bytes), "\n"), nil
}

func (ht *HostsTxt) HostsFile() []byte {
	var returnable []byte
	for _, h := range ht.HostList {
		returnable = append(returnable, []byte(h.String())...)
	}
	return returnable
}

func NewHostsTxt(file string) (*HostsTxt, error) {
	var ht HostsTxt
	hosts, err := ReadHostsFile(file)
	ht.hostMap = make(map[string]string)
	if err != nil {
		return nil, err
	}
	for _, v := range hosts {
		spl := strings.SplitN(v, "=", 2)
		if len(spl) == 2 {
			ht.HostList = append(ht.HostList, Host{Host: spl[0], Destination: spl[1]})
		}
	}
	return &ht, nil
}
