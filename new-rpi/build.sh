#!/bin/bash

GOOS=linux GOARCH=arm GOARM=5 go build -o build/rpio-stuff

docker buildx build \
    --output type=docker \
    --platform linux/arm/v6 \
    -t rpi-gate-controller:latest .

docker save rpi-gate-controller:latest > build/rpi-gate-controller.tar