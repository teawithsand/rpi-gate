package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/r3labs/sse/v2"
)

func RunServer() {
	server := sse.New()
	server.EventTTL = time.Second * 5
	server.AutoReplay = true
	server.CreateStream(sseID)

	router := mux.NewRouter()
	router.Path("/listen").Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("Got client on /listen path")
		server.ServeHTTP(w, r)
	}))
	router.Path("/open-gate").Methods("POST", "GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("Publishing open gate")
		server.Publish(sseID, makeEvent(ServerMessage{
			EventType: EventTypeOpenGate,
		}))
	})
	router.Path("/light-button").Methods("POST", "GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("Publishing light button")
		server.Publish(sseID, makeEvent(ServerMessage{
			EventType: EventTypeLightButton,
		}))
	})

	byteKey := []byte(key)

	srv := http.Server{
		TLSConfig: makeTLSConfig(false),
		Addr:      fmt.Sprintf(":%d", serverPort),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = byteKey
			// Not needed as now we use tls client auth instead
			// givenKey := []byte(r.Header.Get(keyHeader))
			// if !hmac.Equal(byteKey, givenKey) {
			// 	// on key mismatch
			// 	// refuse to handle request
			// 	return
			// }

			router.ServeHTTP(w, r)
		}),
	}
	srv.IdleTimeout = time.Second * 30

	l, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		panic(err)
	}
	tlsListener := tls.NewListener(l, srv.TLSConfig)

	log.Println("Just about to listen...")
	err = srv.Serve(tlsListener)
	panic(err)
}
