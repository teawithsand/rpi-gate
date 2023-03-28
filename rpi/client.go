package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/r3labs/sse/v2"
	"gopkg.in/cenkalti/backoff.v1"
)

type closerFunc func() (err error)

func (cf closerFunc) Close() (err error) {
	return cf()
}

type eventType int

const EventTypeLightButton eventType = 1
const EventTypeOpenGate eventType = 2

type ServerMessage struct {
	EventType eventType `json:"eventType"`
}

func makeEvent(msg ServerMessage) (event *sse.Event) {
	res, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	event = &sse.Event{
		Data: res,
	}
	return
}

type keyTransport struct {
	http.RoundTripper
}

func (ct *keyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// req.Header.Add(keyHeader, key)
	return ct.RoundTripper.RoundTrip(req)
}

func InitRemoteClient(ctx context.Context) (res <-chan eventType) {
	resChan := make(chan eventType)
	res = resChan

	tlsConfig := makeTLSConfig(true)

	transport := &keyTransport{
		RoundTripper: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	go func() {
		client := sse.NewClient("https://gate.teawithsand.com:1997/listen", func(c *sse.Client) {
			c.Connection = &http.Client{
				Transport: transport,
				Timeout:   time.Minute * 30, // required for SSE I guess
			}

			bo := backoff.NewConstantBackOff(time.Second * 5)
			c.ReconnectStrategy = bo
		})

		for {
			log.Println("Initializing message listener...")
			err := client.SubscribeWithContext(ctx, sseID, func(msg *sse.Event) {
				if len(msg.Data) == 0 {
					return
				}

				var decoded ServerMessage
				err := json.Unmarshal(msg.Data, &decoded)

				if err != nil {
					log.Printf("Filed to decode message from remote!")
					return
				}

				resChan <- decoded.EventType
			})

			if errors.Is(err, context.DeadlineExceeded) {
				return
			}

			if err != nil {
				log.Println("Client listener filed! Retrying in 5 minutes...", err)
				time.Sleep(5 * time.Minute)
			}
		}
	}()

	return
}
