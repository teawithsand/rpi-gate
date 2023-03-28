#!/bin/bash

GOOS=linux GOARCH=arm GOARM=5 go build -o client
go build -o server
/build/deployer.py rpi-gate-install server client