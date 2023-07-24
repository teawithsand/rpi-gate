package main

import (
	"sync"
	"time"

	"github.com/warthog618/gpiod"
	"github.com/warthog618/gpiod/device/rpi"
)

type GateState struct {
	OpenDeadline time.Time
}

type GateControls struct {
	lock             *sync.Mutex
	closers          []func() error
	gateState        GateState
	gatePin          *gpiod.Line
	refreshStateChan chan struct{}
}

func (c *GateControls) Close() (err error) {
	for _, closer := range c.closers {
		if err == nil {
			err = closer()
		} else {
			closer()
		}
	}

	return
}

func (c *GateControls) innerOpenGate() {
	c.gatePin.SetValue(1)
	time.Sleep(time.Second)
	c.gatePin.SetValue(0)
}

func (c *GateControls) OpenGate() {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.innerOpenGate()
}

func (c *GateControls) UpdateState(state GateState) {
	c.gateState = state
	c.refreshStateChan <- struct{}{}
}

func InitGate(c *gpiod.Chip) (controls *GateControls, err error) {
	controls = &GateControls{
		lock: &sync.Mutex{},
	}

	defer func() {
		if err != nil {
			controls.Close()
		}
	}()

	gatePin, err := c.RequestLine(rpi.GPIO23, gpiod.AsOutput(1))
	if err != nil {
		return
	}
	controls.gatePin = gatePin

	gatePin.SetValue(0)

	{
		gateCheckerCloseChan := make(chan struct{})
		controls.closers = append(controls.closers, func() error {
			gateCheckerCloseChan <- struct{}{}
			close(gateCheckerCloseChan)
			return nil
		})
		go func() {
			ticker := time.NewTicker(time.Minute - time.Second*2)
			defer ticker.Stop()

			check := func() {
				defer controls.lock.Unlock()
				controls.lock.Lock()

				now := time.Now()
				if controls.gateState.OpenDeadline.Compare(now) > 0 {
					controls.innerOpenGate()
				}
			}

			for {
				select {
				case <-ticker.C:
					check()
				case <-controls.refreshStateChan:
					check()
				case <-gateCheckerCloseChan:
					return
				}
			}
		}()
	}
	return
}
