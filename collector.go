package main

import (
	"fmt"

	"strconv"

	"sync"

	"github.com/cakturk/go-netstat/netstat"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// SocketSetExporter Exporter of the status of connection
type SocketSetExporter struct {
	socketStatusMetrics *prometheus.GaugeVec
	socketCountMetrics  *prometheus.GaugeVec
	mutex               sync.Mutex
	sockets             *socketSet
	logger              log.Logger
	debug               bool
}

var SocketSetLabels = []string{"name", "srchost", "srcport", "dsthost", "dstport", "protocol", "status", "process"}

// NewSocketSetExporter Creator of SocketSetExporter
func NewSocketSetExporter(sockets *socketSet, logger log.Logger, debug bool) *SocketSetExporter {

	return &SocketSetExporter{
		socketStatusMetrics: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "connection_status_up",
				Help: "Connection status of the socket (0 down - 1 up).",
			}, SocketSetLabels),
		socketCountMetrics: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "connection_status_count",
				Help: "number of socket with same parameter.",
			}, SocketSetLabels),
		sockets: sockets,
		logger:  logger,
		debug:   debug,
	}
}

// Describe Implements interface
func (exporter *SocketSetExporter) Describe(prometheusChannel chan<- *prometheus.Desc) {
	exporter.socketStatusMetrics.Describe(prometheusChannel)
	return
}

// Collect Implements interface
func (exporter *SocketSetExporter) Collect(prometheusChannel chan<- prometheus.Metric) {
	exporter.mutex.Lock()
	defer exporter.mutex.Unlock()
	exporter.sockets.collect(exporter)
	exporter.socketStatusMetrics.Collect(prometheusChannel)
	exporter.socketCountMetrics.Collect(prometheusChannel)
	return
}

// Calls the method collect of each socket in the socketSet
func (thisSocketSet *socketSet) collect(exporter *SocketSetExporter) {

	for proto, sockets := range thisSocketSet.socksByType {
		var fn func(accept netstat.AcceptFn) ([]netstat.SockTabEntry, error)
		if proto == "tcp" {
			fn = netstat.TCPSocks
		} else if proto == "udp" {
			fn = netstat.UDPSocks
		} else if proto == "tcp6" {
			fn = netstat.TCP6Socks
		} else if proto == "udp6" {
			fn = netstat.UDP6Socks
		}
		entries, err := fn(netstat.NoopFilter)

		if err != nil {
			level.Error(exporter.logger).Log("msg", fmt.Sprintf("%+v", err))
		}

		for _, currentSocket := range sockets {
			currentSocket.collect(exporter, entries, proto)
		}

	}
	// if len(thisSocketSet.tcpSockets) > 0 {
	// 	entries, err := netstat.TCPSocks(netstat.NoopFilter)
	// 	if err != nil {
	// 		level.Error(exporter.logger).Log("msg", fmt.Sprintf("%+v", err))
	// 	}

	// 	for _, currentSocket := range thisSocketSet.tcpSockets {
	// 		currentSocket.collect(exporter, entries)
	// 	}
	// }
	// if len(thisSocketSet.udpSockets) > 0 {
	// 	entries, err := netstat.UDPSocks(netstat.NoopFilter)
	// 	if err != nil {
	// 		level.Error(exporter.logger).Log("msg", fmt.Sprintf("%+v", err))
	// 	}

	// 	for _, currentSocket := range thisSocketSet.udpSockets {
	// 		currentSocket.collect(exporter, entries)
	// 	}
	// }
	return
}

// Checks the status of the connection of a socket and updates it in the Metric
func (thisSocket *socket) collect(exporter *SocketSetExporter, entries []netstat.SockTabEntry, proto string) {
	connectionCount := 0

	for _, entry := range entries {
		if exporter.debug {
			level.Debug(exporter.logger).Log("debug", "check", "cur_socket", fmt.Sprintf("%+v", entry))
			level.Debug(exporter.logger).Log("debug", "check", "check", thisSocket.ToString())
		}
		if thisSocket.Status == "listen" && entry.State != netstat.Listen {
			if exporter.debug {
				level.Debug(exporter.logger).Log("debug", "check", "msg", "wrong cnx status (!= listen)")
			}
			continue
		}
		if thisSocket.Status == "established" && entry.State != netstat.Established {
			if exporter.debug {
				level.Debug(exporter.logger).Log("debug", "check", "msg", "wrong cnx status (!= established)")
			}
			continue
		}
		if !thisSocket.ip_src.Equal(entry.LocalAddr.IP) {
			if exporter.debug {
				level.Debug(exporter.logger).Log("debug", "check", "msg", "wrong source addr")
			}
			continue
		}
		if !thisSocket.ip_dst.Equal(entry.RemoteAddr.IP) {
			if exporter.debug {
				level.Debug(exporter.logger).Log("debug", "check", "msg", "wrong destination addr")
			}
			continue
		}
		// if thisSocket.SrcHost != "" {
		// 	if strings.EqualFold(thisSocket.SrcHost, "any") || thisSocket.SrcHost == "*" {
		// 		if proto == "tcp6" || proto == "udp6" {
		// 			if entry.LocalAddr.IP.String() != "::" {
		// 				continue
		// 			}
		// 		} else {
		// 			if entry.LocalAddr.IP.String() != "0.0.0.0" {
		// 				continue
		// 			}

		// 		}
		// 	} else if thisSocket.SrcHost != entry.LocalAddr.IP.String() {
		// 		continue
		// 	}
		// }
		// if thisSocket.DstHost != "" {
		// 	if strings.EqualFold(thisSocket.DstHost, "any") || thisSocket.DstHost == "*" {
		// 		if proto == "tcp6" || proto == "udp6" {
		// 			if entry.RemoteAddr.IP.String() != "::" {
		// 				continue
		// 			}
		// 		} else {
		// 			if entry.RemoteAddr.IP.String() != "0.0.0.0" {
		// 				continue
		// 			}

		// 		}
		// 	} else if thisSocket.DstHost != entry.RemoteAddr.IP.String() {
		// 		continue
		// 	}
		// }
		if thisSocket.srcPort != 0 && thisSocket.srcPort != entry.LocalAddr.Port {
			if exporter.debug {
				level.Debug(exporter.logger).Log("debug", "check", "msg", "wrong source port")
			}
			continue
		}
		if thisSocket.dstPort != 0 && thisSocket.dstPort != entry.RemoteAddr.Port {
			if exporter.debug {
				level.Debug(exporter.logger).Log("debug", "check", "msg", "wrong destination port")
			}
			continue
		}
		if thisSocket.ProcessName != "" && !thisSocket.procPattern.MatchString(entry.Process.Name) {
			if exporter.debug {
				level.Debug(exporter.logger).Log("debug", "check", "msg", "wrong local processName.")
			}
			continue
		}

		connectionCount++
		if exporter.debug {
			level.Debug(exporter.logger).Log("debug", "check", "msg", fmt.Sprintf("ok count=%d", connectionCount))
		}
		// break
	}
	labels := make([]string, len(SocketSetLabels))
	labels[0] = thisSocket.Name
	if thisSocket.SrcHost == "" {
		labels[1] = "*"
	} else {
		labels[1] = thisSocket.SrcHost
	}
	if thisSocket.srcPort == 0 {
		labels[2] = "*"
	} else {
		labels[2] = strconv.Itoa(int(thisSocket.srcPort))
	}
	if thisSocket.DstHost == "" {
		labels[3] = "*"
	} else {
		labels[3] = thisSocket.DstHost
	}
	if thisSocket.dstPort == 0 {
		labels[4] = "*"
	} else {
		labels[4] = strconv.Itoa(int(thisSocket.dstPort))
	}
	labels[5] = thisSocket.Protocol
	labels[6] = thisSocket.Status

	labels[7] = thisSocket.ProcessName

	connectionStatus := 0
	if connectionCount > 0 {
		connectionStatus = 1
	}
	// Updated the status of the socket in the metric
	exporter.socketStatusMetrics.WithLabelValues(labels[:]...).Set(float64(connectionStatus))

	exporter.socketCountMetrics.WithLabelValues(labels[:]...).Set(float64(connectionCount))
	level.Debug(exporter.logger).Log("status", fmt.Sprintf("%d", connectionStatus),
		"count", fmt.Sprintf("%d", connectionCount),
		"labels", fmt.Sprintf("%+q", labels))

	return
}
