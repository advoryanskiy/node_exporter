package collector

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	ovpnSockets = kingpin.Flag("collector.openvpn.sockets", "Unix socket files to read metrics from. Format: label1:/file1,label2:/file2,...,labelN:/fileN").Default("").String()
)

type ovpnCollector struct {
	sockets map[string]string

	upDesc               *prometheus.Desc
	statusUpdateTimeDesc *prometheus.Desc

	serverUpSinceDesc *prometheus.Desc

	clientsNumberDesc *prometheus.Desc
	bytesInDesc       *prometheus.Desc
	bytesOutDesc      *prometheus.Desc
}

func init() {
	Factories['openvpn'] = NewOpenVPNCollector
}

func NewOpenVPNCollector() (Collector, error) {
	sockets := map[string]string{}

	// parse sockets
	servers := strings.Split(ovpnSockets, ",")
	for server := range servers {
		parts := strings.Split(server, ":")
		if len(parts) != 2 {
			panic("Failed to configure openvpn collector")
		}

		sockets[parts[0]] = parts[1]
	}

	return &ovpnCollector{
		sockets: sockets,
		upDesc: prometheus.NewDesc(
			prometheus.BuildFQName("openvpn", "", "up"),
			"Whether scraping OpenVPN's metrics was successful.",
			[]string{"vpn_label"},
		),
		statusUpdateTimeDesc: prometheus.NewDesc(
			prometheus.BuildFQName("openvpn", "", "status_update_time_seconds"),
			"UNIX timestamp at which the OpenVPN statistics were updated.",
			[]string{"vpn_label"},
		),
		serverUpSinceDesc: prometheus.NewDesc(
			prometheus.BuildFQName("openvpn", "", "up_since_time_seconds"),
			"UNIX timestamp at which the OpenVPN server were started.",
			[]string{"vpn_label"},
		),
		clientsNumberDesc: prometheus.NewDesc(
			prometheus.BuildFQName("openvpn", "", "clients_number"),
			"Total connected clients count."
			[]string{"vpn_label"},
		),
		bytesInDesc: prometheus.NewDesc(
			prometheus.BuildFQName("openvpn", "", "bytes_in"),
			"Total received bytes.",
			[]string{"vpn_label"},
		),
		bytesOutDesc: prometheus.NewDesc(
			prometheus.BuildFQName("openvpn", "", "bytes_out"),
			"Total sent bytes.",
			[]string{"vpn_label"},
		),
	}, nil
}
