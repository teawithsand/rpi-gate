package main

import (
	"log"
	"sync"
	"time"

	"github.com/warthog618/gpiod"
	"github.com/warthog618/gpiod/device/rpi"
)

const baseDuration = time.Minute * 5
const incByDuration = time.Minute * 10
const initializingDuration = time.Second * 5

type lightsButtonStateEnum int

const stateInitializing = 1
const stateListening = 0

type LightsState struct {
	LightsDeadline time.Time
}

type LightControls struct {
	lock            *sync.Mutex
	closers         []func() error
	lightsStateChan chan LightsState

	lightsPin *gpiod.Line
	buttonPin *gpiod.Line
}

func (c *LightControls) Close() (err error) {
	for _, closer := range c.closers {
		if err == nil {
			err = closer()
		} else {
			closer()
		}
	}

	return
}

func (c *LightControls) UpdateState(state LightsState) {
	c.lightsStateChan <- state
}

func InitLights(c *gpiod.Chip) (controls *LightControls, err error) {
	controls = &LightControls{
		lock:            &sync.Mutex{},
		lightsStateChan: make(chan LightsState),
	}

	defer func() {
		if err != nil {
			controls.Close()
		}
	}()

	lightsPin, err := c.RequestLine(rpi.GPIO27, gpiod.AsOutput(1))
	if err != nil {
		return
	}
	controls.lightsPin = lightsPin
	err = lightsPin.SetValue(1)
	if err != nil {
		return
	}

	inPin, err := c.RequestLine(rpi.GPIO17, gpiod.AsInput, gpiod.WithPullDown)
	controls.buttonPin = inPin

	buttonChan := make(chan struct{})

	stateLock := &sync.Mutex{}
	state := LightsState{}

	{
		controls.closers = append(controls.closers, func() error {
			close(controls.lightsStateChan)
			return nil
		})

		updateTimer := time.NewTimer(0)

		reapplyState := func() {
			stateLock.Lock()
			defer stateLock.Unlock()

			now := time.Now()
			if now.Compare(state.LightsDeadline) < 0 {
				delta := state.LightsDeadline.Sub(now)
				updateTimer.Reset(delta + time.Millisecond)

				err := controls.lightsPin.SetValue(0) // lights enabled is 0
				if err != nil {
					log.Println("Filed to enable lights", err)
				}
			} else {
				err := controls.lightsPin.SetValue(1)
				if err != nil {
					log.Println("Filed to disable lights", err)
				}
			}
		}

		go func() {
			for {
				select {
				case <-updateTimer.C:
					reapplyState()
				case newState, ok := <-controls.lightsStateChan:
					if !ok {
						return
					}

					stateLock.Lock()
					state = newState
					stateLock.Unlock()
					reapplyState()
				}
			}
		}()
	}

	{
		buttonCloseChan := make(chan struct{})
		controls.closers = append(controls.closers, func() error {
			buttonCloseChan <- struct{}{}
			close(buttonCloseChan)
			return nil
		})
		buttonPressed := false
		go func() {
			defer close(buttonChan)
			for {
				select {
				case <-buttonCloseChan:
					break
				default:
				}
				state, err := inPin.Value()
				if err != nil {
					log.Println("Got error when reading in pin value:", err)
					continue
				}

				if !buttonPressed && state > 0 {
					buttonPressed = true
					buttonChan <- struct{}{}
				} else {
					buttonPressed = false
				}
				time.Sleep(time.Second / 4)
			}
		}()

		go func() {
			var buttonDuration time.Duration
			onButtonPress := func() {
				stateLock.Lock()
				defer stateLock.Unlock()

				isPendingTimeAdding := false

				isEnabled := state.LightsDeadline.IsZero()

				if isEnabled && !isPendingTimeAdding {
					controls.lightsStateChan <- LightsState{
						LightsDeadline: time.Time{},
					}
				} else if isPendingTimeAdding {
					buttonDuration += incByDuration
				} else {
					now := time.Now()
					controls.lightsStateChan <- LightsState{
						LightsDeadline: now.Add(initializingDuration + time.Second),
					}

					buttonDuration = baseDuration
					isPendingTimeAdding = true

					// we can leak that goroutine
					// though it's kind of unsound and may crash program if lightsStateChan is closed
					// but this program is unsound anyway so who cares
					// not me
					// and I can use docker to auto restart on crashes
					log.Println("Initialized button time adding")
					go func() {
						stateLock.Lock()
						defer stateLock.Unlock()

						time.Sleep(initializingDuration)

						controls.lightsStateChan <- LightsState{
							LightsDeadline: now.Add(buttonDuration),
						}

						log.Println("Done button time adding; Lights are running for", buttonDuration.Seconds(), "seconds")
						isPendingTimeAdding = false
					}()
				}
			}
			for {
				select {
				case <-buttonCloseChan:
					return
				case _, ok := <-buttonChan:
					if !ok {
						return
					}

					onButtonPress()
				}
			}
		}()
	}

	return
}
