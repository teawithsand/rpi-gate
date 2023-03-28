package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"
)

const baseDuration = time.Minute * 5
const incByDuration = time.Minute * 10
const icingDuration = time.Second * 5

type StateEnum int

const StateDisabled = 0
const StateInitializing = 1
const StateEnabled = 2

type state struct {
	State             StateEnum
	DisableTimer      *time.Timer
	InitializingTimer *time.Timer

	RunForDuration time.Duration
}

func main() {
	mode := os.Getenv("TWS_RUN_MODE")
	if len(mode) == 0 || strings.EqualFold(mode, "normal") {

	} else if strings.EqualFold(mode, "server") {
		RunServer()
		return
	} else if strings.EqualFold(mode, "clientdebug") {
		dataChan := InitRemoteClient(context.Background())
		for v := range dataChan {
			log.Println("Received value", v)
		}
		return
	}
	actions, err := initRPIO()
	if err != nil {
		panic(err)
	}
	defer actions.Close()

	state := state{
		InitializingTimer: time.NewTimer(0),
		DisableTimer:      time.NewTimer(0),
	}
	time.Sleep(time.Second)
	select {
	case <-state.InitializingTimer.C:
	default:
	}
	select {
	case <-state.DisableTimer.C:
	default:
	}

	remoteChan := InitRemoteClient(context.Background())

	onButtonPressed := func() {
		if state.State == StateDisabled {
			enabled = true
			syncState()

			state.DisableTimer.Stop()

			state.State = StateInitializing
			state.RunForDuration = baseDuration
			state.InitializingTimer = time.NewTimer(icingDuration)
		} else if state.State == StateInitializing {
			state.RunForDuration += incByDuration
		} else if state.State == StateEnabled {
			state.DisableTimer.Stop()

			enabled = false
			state.State = StateDisabled
			syncState()
		}
	}

	for {
		log.Printf("BEFORE %+#v\n", state)
		select {
		case val := <-remoteChan:
			log.Println("Received value", val)
			if val == EventTypeLightButton {
				log.Println("Enabling lights")
				onButtonPressed()
			} else if val == EventTypeOpenGate {
				log.Println("Opening gate")
				go actions.OpenGate()
			}
		case <-buttonChan:
			onButtonPressed()
		case <-state.InitializingTimer.C:
			state.State = StateEnabled
			state.DisableTimer = time.NewTimer(state.RunForDuration)
		case <-state.DisableTimer.C:
			enabled = false
			state.State = StateDisabled
			syncState()
		}
		log.Printf("AFTER %+#v\n", state)
	}
}
