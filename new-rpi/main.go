package main

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/warthog618/gpiod"
)

func main() {
	c, err := gpiod.NewChip("gpiochip0", gpiod.WithConsumer("lights"))
	if err != nil {
		return
	}

	gate, err := InitGate(c)
	if err != nil {
		panic(err)
	}

	lights, err := InitLights(c)
	if err != nil {
		panic(err)
	}

	http.Handle("/gate/open", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer w.WriteHeader(200)

		go func() {
			log.Println("Pre open gate")
			log.Println("Post open gate")
			gate.OpenGate()
		}()
	}))
	http.Handle("/gate/keep-open", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer w.WriteHeader(200)

		durationSecondsRaw := r.URL.Query().Get("duration-seconds")
		parsed, err := strconv.Atoi(durationSecondsRaw)
		if err != nil || parsed < 0 || parsed > int(time.Hour*24*7/time.Second) {
			return
		}
		gate.UpdateState(GateState{
			OpenDeadline: time.Now().Add(time.Duration(parsed) * time.Second),
		})
	}))
	http.Handle("/gate/close", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer w.WriteHeader(200)

		gate.UpdateState(GateState{
			OpenDeadline: time.Time{},
		})
	}))
	http.Handle("/lights/on", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer w.WriteHeader(200)

		durationSecondsRaw := r.URL.Query().Get("duration-seconds")
		parsed, err := strconv.Atoi(durationSecondsRaw)
		if err != nil || parsed < 0 || parsed > int(time.Hour*24*7/time.Second) {
			return
		}

		lights.UpdateState(LightsState{
			LightsDeadline: time.Now().Add(time.Duration(parsed) * time.Second),
		})
	}))
	http.Handle("/lights/off", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer w.WriteHeader(200)

		lights.UpdateState(LightsState{
			LightsDeadline: time.Time{},
		})
	}))

	log.Println("Stuff initialized; we're about to start HTTP server")
	err = http.ListenAndServe(":8080", nil)
	panic(err)
}
