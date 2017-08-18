#!/bin/sh

./node_exporter --collector.openvpn.sockets="tcp:/Users/alexey/Documents/Projects/tmp/ovpn/tcp.sock,udp:/Users/alexey/Documents/Projects/tmp/ovpn/udp.sock"
