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

var addr = flag.String("connection", "localhost:8090", "http service addr")

func main() {
	flag.Parse()
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/echo"}
	log.Printf("Try to connect to %s", u.String())

	CC := centralnodeconnection.CentralNodeConnection{
		Secret:         "ownsecret",
		ResponseSecret: "ownsecret2",
		Url:            u,
	}

	hub2 := hub.NewHub(&CC)

	go hub2.Start()

	(&worker.DockerWorker{Hub: hub2}).Start()
}