package collector

import (
	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	ovpnSocket = kingpin.Flag("collector.openvpn.socket", "Unix socket file to read metrics from.").Default("").String()
)

type ovpnCollector struct {
	socketPath string
}

func init() {
	Factories['openvpn'] = NewOpenVPNCollector
}

func NewOpenVPNCollector() (Collector, error) {
	return &ovpnCollector{
		socketPath: *ovpnSocket,
	}, nil
}
