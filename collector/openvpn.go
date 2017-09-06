// Copyright 2015 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build !noovpn

package collector

import (
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	ovpnSockets = kingpin.Flag("collector.openvpn.sockets", "Unix socket files to read metrics from. Format: label1:/file1,label2:/file2,...,labelN:/fileN").Default("").String()
)

type ovpnCollector struct {
	sockets map[string]string

	upDesc            *prometheus.Desc
	serverUpSinceDesc *prometheus.Desc

	clientsNumberDesc *prometheus.Desc
	bytesInDesc       *prometheus.Desc
	bytesOutDesc      *prometheus.Desc
}

func init() {
	Factories["openvpn"] = NewOpenVPNCollector
}

func NewOpenVPNCollector() (Collector, error) {
	sockets := map[string]string{}

	// parse sockets
	servers := strings.Split(*ovpnSockets, ",")
	for _, server := range servers {
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
			[]string{"vpn_label"}, nil,
		),
		serverUpSinceDesc: prometheus.NewDesc(
			prometheus.BuildFQName("openvpn", "", "up_since_time_seconds"),
			"UNIX timestamp at which the OpenVPN server were started.",
			[]string{"vpn_label"}, nil,
		),
		clientsNumberDesc: prometheus.NewDesc(
			prometheus.BuildFQName("openvpn", "", "clients_number"),
			"Total connected clients count.",
			[]string{"vpn_label"}, nil,
		),
		bytesInDesc: prometheus.NewDesc(
			prometheus.BuildFQName("openvpn", "", "bytes_in"),
			"Total received bytes.",
			[]string{"vpn_label"}, nil,
		),
		bytesOutDesc: prometheus.NewDesc(
			prometheus.BuildFQName("openvpn", "", "bytes_out"),
			"Total sent bytes.",
			[]string{"vpn_label"}, nil,
		),
	}, nil
}

func (c *ovpnCollector) collectMetrics(label, socket string, ch chan<- prometheus.Metric) error {
	conn, err := net.DialTimeout("unix", socket, 3*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	state, err := c.sendCommand(conn, "state\n")
	if err != nil {
		return err
	}
	if err = c.publishState(state, label, ch); err != nil {
		return err
	}

	stats, err := c.sendCommand(conn, "load-stats\n")
	if err != nil {
		return err
	}
	if err = c.publishStats(stats, label, ch); err != nil {
		return err
	}

	return nil
}

func (c *ovpnCollector) sendCommand(conn net.Conn, cmd string) (string, error) {
	if _, err := conn.Write([]byte(cmd)); err != nil {
		return "", err
	}

	data := ""
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf[:])
		if err != nil {
			return "", err
		}

		str := string(buf[0:n])
		str = strings.Replace(str, ">INFO(.)*\r\n", "", -1)

		data += str
		if cmd == "load-stats\n" && len(data) > 0 {
			break
		} else if strings.HasSuffix(data, "\nEND\r\n") {
			break
		}
	}

	return data, nil
}

func (c *ovpnCollector) publishState(state, label string, ch chan<- prometheus.Metric) error {
	for _, line := range strings.Split(state, "\n") {
		if len(strings.TrimSpace(line)) == 0 {
			continue
		}

		parts := strings.Split(line, ",")
		if strings.HasPrefix(parts[0], ">INFO") ||
			strings.HasPrefix(parts[0], "END") ||
			strings.HasPrefix(parts[0], ">CLIENT") {
			continue
		} else {
			upSince, err := strconv.Atoi(parts[0])
			if err != nil {
				return err
			}

			ch <- prometheus.MustNewConstMetric(
				c.serverUpSinceDesc,
				prometheus.GaugeValue,
				float64(upSince),
				label,
			)
		}
	}

	return nil
}

func (c *ovpnCollector) publishStats(stats, label string, ch chan<- prometheus.Metric) error {
	line := strings.Replace(stats, "SUCCESS: ", "", -1)
	parts := strings.Split(line, ",")

	// parse clients count
	clients, err := strconv.Atoi(strings.Replace(parts[0], "nclients=", "", -1))
	if err != nil {
		return err
	}

	ch <- prometheus.MustNewConstMetric(
		c.clientsNumberDesc,
		prometheus.GaugeValue,
		float64(clients),
		label,
	)

	// parse received bytes count
	bytesIn, err := strconv.ParseInt(
		strings.Replace(parts[1], "bytesin=", "", -1),
		10, 64,
	)
	if err != nil {
		return err
	}

	ch <- prometheus.MustNewConstMetric(
		c.bytesInDesc,
		prometheus.GaugeValue,
		float64(bytesIn),
		label,
	)

	// parse sent bytes count
	bytesOut, err := strconv.ParseInt(strings.TrimSpace(
		strings.Replace(parts[2], "bytesout=", "", -1),
	), 10, 64)
	if err != nil {
		return err
	}

	ch <- prometheus.MustNewConstMetric(
		c.bytesOutDesc,
		prometheus.GaugeValue,
		float64(bytesOut),
		label,
	)

	return nil
}

func (c *ovpnCollector) Update(ch chan<- prometheus.Metric) error {
	for label, socket := range c.sockets {
		err := c.collectMetrics(label, socket, ch)
		if err == nil {
			ch <- prometheus.MustNewConstMetric(c.upDesc, prometheus.GaugeValue, 1.0, label)
		} else {
			ch <- prometheus.MustNewConstMetric(c.upDesc, prometheus.GaugeValue, 0.0, label)
		}
	}
	return nil
}
