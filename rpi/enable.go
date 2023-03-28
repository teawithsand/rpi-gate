package main

import (
	"io"
	"sync"
	"time"

	"github.com/warthog618/gpiod"
	"github.com/warthog618/gpiod/device/rpi"
)

var enabled bool
var enabledLock = &sync.Mutex{}
var buttonChan = make(chan struct{})
var syncState func()

func Must(err error) {
	if err != nil {
		panic(err)
	}
}

type Actions interface {
	io.Closer

	OpenGate()
}

type actionsImpl struct {
	io.Closer
	lock *sync.Mutex
	line *gpiod.Line
}

func (a *actionsImpl) OpenGate() {
	a.lock.Lock()
	defer a.lock.Unlock()

	a.line.SetValue(1)
	time.Sleep(time.Second)
	a.line.SetValue(0)
}

func initRPIO() (res Actions, err error) {
	c, err := gpiod.NewChip("gpiochip0", gpiod.WithConsumer("lights"))
	if err != nil {
		return
	}

	innerRes := &actionsImpl{
		Closer: c,
		lock:   &sync.Mutex{},
	}

	res = innerRes

	defer func() {
		if err != nil {
			res.Close()
		}
	}()

	outPin, err := c.RequestLine(rpi.GPIO27, gpiod.AsOutput(1))
	if err != nil {
		return
	}

	gatePin, err := c.RequestLine(rpi.GPIO23, gpiod.AsOutput(1))
	if err != nil {
		return
	}
	innerRes.line = gatePin

	gatePin.SetValue(0)

	inPin, err := c.RequestLine(rpi.GPIO17, gpiod.AsInput, gpiod.WithPullDown)
	if err != nil {
		return
	}

	syncState = func() {
		enabledLock.Lock()
		defer enabledLock.Unlock()

		if enabled {
			Must(outPin.SetValue(0))
		} else {
			Must(outPin.SetValue(1))
		}
	}
	syncState()

	buttonPressed := false
	go func() {
		for {
			state, err := inPin.Value()
			Must(err)

			if !buttonPressed && state > 0 {
				buttonPressed = true
				buttonChan <- struct{}{}
			} else {
				buttonPressed = false
			}
			time.Sleep(time.Second / 4)
		}
	}()

	return
}
