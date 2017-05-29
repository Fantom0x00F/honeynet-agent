package main

import (
	"flag"
	"net/url"
	"log"

	"../centralnodeconnection"
	"../hub"
	"../worker"
	"io/ioutil"
)

var addr = flag.String("node", "localhost", "http service addr")
var socketpath = flag.String("path", "/echo", "endpoint for webscoket connection")
var certFilePath = flag.String("cert", "", "path to base certificate")


func main() {
	flag.Parse()

	rootPEM, err := ioutil.ReadFile(*certFilePath)
	if err != nil {
		log.Println("failed to load base certificate")
		panic(err)
	}

	u := url.URL{Scheme: "wss", Host: *addr, Path: *socketpath}
	log.Printf("Try to connect to %s", u.String())

	CC := centralnodeconnection.NewCentralNodeConnection(u, rootPEM, "ownsecret", "ownsecret2")

	hub2 := hub.NewHub(CC)

	go hub2.Start()

	worker.NewDockerWorker(hub2).Start()
}
