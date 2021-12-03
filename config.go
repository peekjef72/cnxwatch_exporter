package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

type socketSet struct {
	Sockets []socket `yalm:"sockets"`

	// Catches all undefined fields and must be empty after parsing.
	XXX map[string]interface{} `yaml:",inline" json:"-"`

	socksByType map[string][]socket
}

type socket struct {
	Name        string `yaml:"name"`
	Host        string `yaml:"host,omitempty"`
	SrcHost     string `yaml:"srcHost,omitempty"`
	DstHost     string `yaml:"dstHost,omitempty"`
	Port        string `yaml:"port,omitempty"`
	SrcPort     string `yaml:"srcPort,omitempty"`
	DstPort     string `yaml:"dstPort,omitempty"`
	Protocol    string `yaml:"protocol,omitempty"`
	ProcessName string `yaml:"process,omitempty"`

	Status           string `yaml:"status,omitempty"`
	procPattern      *regexp.Regexp
	srcPort, dstPort uint16
	ip_src, ip_dst   net.IP
}

const (
	// Default values for optional parameters of socket
	defaultProtocol string = "tcp"

	// Default values for optional parameters of socket
	defaultStatus string = "listen"
)

// *************************************************************
//
// *************************************************************
// Load attempts to parse the given config file and return a Config object.
func Load(configFile string) (*socketSet, error) {
	//	log.Infof("Loading profiles from %s", profilesFile)
	buf, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	sockets := socketSet{}
	err = yaml.Unmarshal(buf, &sockets)
	if err != nil {
		return nil, err
	}
	err = checkOverflow(sockets.XXX, "sockets")
	if err != nil {
		return nil, err
	}

	err = sockets.check()
	if err != nil {
		return nil, err
	}

	sockets.socksByType = make(map[string][]socket)

	for _, proto := range []string{"tcp", "udp", "tcp6", "udp6"} {
		socks := sockets.getSocketProtocol(proto)
		if len(socks) > 0 {
			sockets.socksByType[proto] = socks
		}
	}

	return &sockets, nil
}

// *************************************************************
//
// socketSet
//
// *************************************************************
// check the sanity of the sockets in the set
func (thisSocketSet *socketSet) check() error {
	for index := range thisSocketSet.Sockets {
		err := thisSocketSet.Sockets[index].check()
		if err != nil {
			return (err)
		}
	}
	return (nil)
}

// collect slice of sockets for a specific protocol
func (thisSocketSet *socketSet) getSocketProtocol(protocol string) []socket {
	list := make([]socket, 0)
	for index := range thisSocketSet.Sockets {
		if thisSocketSet.Sockets[index].Protocol == protocol {
			list = append(list, thisSocketSet.Sockets[index])
		}
	}
	return list
}

// collect slice of sockets for a specific protocol
func (thisSocket *socket) resolveHostname(hostname string) (net.IP, error) {
	var err error
	// var ip_str string

	if hostname == "" {
		hostname = "*"
	}
	if strings.EqualFold(hostname, "any") || hostname == "*" {
		// if thisSocket.Protocol == "tcp6" {
		hostname = "0.0.0.0"
		// } else {
		// ip_str = "0.0.0.0"
		// }
		// } else {
	}
	var ips []net.IP
	var ip net.IP

	ips, err = net.LookupIP(hostname)
	if err != nil {
		return ip, err
	}
	ip = ips[0]
	if thisSocket.Protocol == "tcp" || thisSocket.Protocol == "udp" {
		ip = ip.To4()
	}
	return ip, err
}

// *************************************************************
//
// socket
//
// *************************************************************
// Check the sanity of the socket and fills the default values
func (thisSocket *socket) check() error {
	var err error

	if thisSocket.Name == "" {
		return (fmt.Errorf("socket must have the field name set"))
	}
	if thisSocket.Status == "" {
		thisSocket.Status = defaultStatus
	}
	// Check if the protocol is among the valid ones
	if IsValidStatus(thisSocket.Status) == false {
		return (fmt.Errorf("The status of the socket is not a valid one"))
	}

	if thisSocket.Status == "listen" {
		if thisSocket.SrcHost == "" && thisSocket.Host == "" {
			return (fmt.Errorf("socket must have the field host or srcHost set"))
		}
		if thisSocket.Port == "" && thisSocket.SrcPort == "" {
			return (fmt.Errorf("socket must have the field port or srcPort"))
		}
	} else {
		if thisSocket.SrcHost == "" && thisSocket.Host == "" && thisSocket.DstHost == "" {
			return (fmt.Errorf("socket must have the field host or srcHost or dstHost set"))
		}

		if thisSocket.Port == "" && thisSocket.SrcPort == "" && thisSocket.DstPort == "" {
			return (fmt.Errorf("socket must have the field port or srcPort or dstPort set"))
		}
	}

	if thisSocket.SrcHost == "" {
		thisSocket.SrcHost = thisSocket.Host
	}
	thisSocket.ip_src, err = thisSocket.resolveHostname(thisSocket.SrcHost)
	if err != nil {
		return err
	}
	thisSocket.ip_dst, err = thisSocket.resolveHostname(thisSocket.DstHost)
	if err != nil {
		return err
	}

	if thisSocket.SrcPort == "" {
		thisSocket.SrcPort = thisSocket.Port
	}
	var tmp_port int
	if thisSocket.SrcPort != "" {
		tmp_port, err = strconv.Atoi(thisSocket.SrcPort)
		if err != nil {
			return err
		}
		thisSocket.srcPort = uint16(tmp_port)
	}

	if thisSocket.DstPort != "" {
		tmp_port, err = strconv.Atoi(thisSocket.DstPort)
		if err != nil {
			return err
		}
		thisSocket.dstPort = uint16(tmp_port)
	}

	if thisSocket.Protocol == "" {
		thisSocket.Protocol = defaultProtocol
	}

	// Check if the protocol is among the valid ones
	if IsValidProtocol(thisSocket.Protocol) == false {
		return (fmt.Errorf("The protocol of the socket is not a valid one"))
	}

	// if processName pattern is specified, build a regex pattern

	if thisSocket.ProcessName != "" {
		thisSocket.procPattern, err = regexp.Compile("^" + thisSocket.ProcessName + "$")
		if err != nil {
			return err
		}
	}
	return (nil)
}

// IsValidProtocol Check if a string is among the valid protocols
func IsValidProtocol(protocol string) bool {
	switch protocol {
	case
		"tcp",
		"tcp4",
		"tcp6",
		"udp",
		"udp4",
		"udp6":
		// "ip",
		// "ip4",
		// "ip6",
		// "unix",
		// "unixgram",
		// "unixpacket":
		return true
	}
	return false
}

// IsValidProtocol Check if a string is among the valid protocols
func IsValidStatus(status string) bool {
	switch status {
	case
		"listen",
		"established":
		return true
	}
	return false
}

// to catch unwanted params in config file
func checkOverflow(m map[string]interface{}, ctx string) error {
	if len(m) > 0 {
		var keys []string
		for k := range m {
			keys = append(keys, k)
		}
		return fmt.Errorf("unknown fields in %s: %s", ctx, strings.Join(keys, ", "))
	}
	return nil
}
