package main

import (
	"flag"
	"os"
	"os/signal"
	"net/url"
	"log"

	"../centralnodeconnection"
	"../hub"
	"../worker"
)

var addr = flag.String("connection", "fantom.com:8090", "http service addr")

func main() {
	rootPEM := []byte(`-----BEGIN CERTIFICATE-----
MIICMzCCAbmgAwIBAgIJAKYvgRMm365FMAoGCCqGSM49BAMCME8xCzAJBgNVBAYT
AlJVMQ8wDQYDVQQIDAZTYW1hcmExDzANBgNVBAcMBlNhbWFyYTEOMAwGA1UECgwF
SG9uZXkxDjAMBgNVBAsMBUhvbmV5MB4XDTE3MDUyOTE3MDUyM1oXDTQ0MTAxNDE3
MDUyM1owTzELMAkGA1UEBhMCUlUxDzANBgNVBAgMBlNhbWFyYTEPMA0GA1UEBwwG
U2FtYXJhMQ4wDAYDVQQKDAVIb25leTEOMAwGA1UECwwFSG9uZXkwdjAQBgcqhkjO
PQIBBgUrgQQAIgNiAATQST8ReWZhCXRGBbMj5ARNFjck9/a+zEZCxaEOkLPJB5IM
UR4frZXB4qPjaQCMKlkELPp+GUaZhM5hBQ/HFDipAptYdIEEMpnz3lO/HiAQYloQ
p0vUG3Yi3EY93qIsrkejYTBfMB0GA1UdDgQWBBTNKR0+bO6Ovb8a7bAx5xTwg+I5
4DAfBgNVHSMEGDAWgBTNKR0+bO6Ovb8a7bAx5xTwg+I54DAMBgNVHRMEBTADAQH/
MA8GA1UdEQQIMAaHBAoAAAMwCgYIKoZIzj0EAwIDaAAwZQIxAPDYPGUk7qxphwVe
J5UbdKDNJlp7R+4QyiBdVOJhlk2YOKWPebI4dbSJqSPvsx7VhgIwCOLU36J6Mvmh
j1MyQX5dDUcKnzFWk4zl80XX+o7N+Xwj7M6G394lhQJJYq3bt3/E
-----END CERTIFICATE-----`)

	flag.Parse()
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "wss", Host: *addr, Path: "/echo"}
	log.Printf("Try to connect to %s", u.String())

	CC := centralnodeconnection.NewCentralNodeConnection(u, rootPEM, "ownsecret", "ownsecret2")

	hub2 := hub.NewHub(CC)

	go hub2.Start()

	worker.NewDockerWorker(hub2).Start()
}
