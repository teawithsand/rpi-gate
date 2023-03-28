import RPi.GPIO as GPIO
import time
import random

OUT_PIN = 27
IN_PIN = 17

GPIO.setmode(GPIO.BCM)
GPIO.setup(OUT_PIN, GPIO.OUT)
GPIO.setup(IN_PIN, GPIO.IN, GPIO.PUD_DOWN)

def on_button_pressed(*args, **kwargs):
	print("Button pressed!")

#print("Started button detection")
#GPIO.add_event_detect(IN_PIN, GPIO.RISING, callback=on_button_pressed)

enabled = False
def disable():
    global enabled
    GPIO.output(OUT_PIN, 1)
    enabled = False

def enable():
    global enabled
    GPIO.output(OUT_PIN, 0)
    enabled = True

def toggle():
    global enabled
    if enabled:
        disable()
    else:
        enable()

disable()

was_button_pressed = False
while True:
    button_pressed = (GPIO.input(IN_PIN) == True)
    if not was_button_pressed and button_pressed:
        print("Button pressed")
        toggle()
        time.sleep(1)

    was_button_pressed = button_pressed
            



