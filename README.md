[![Go Report Card](https://goreportcard.com/badge/github.com/kostiamol/fridgems)](https://goreportcard.com/report/github.com/kostiamol/fridgems)
[![Coverage Status](https://coveralls.io/repos/github/kostiamol/fridgems/badge.svg?branch=master)](https://coveralls.io/github/kostiamol/fridgems?branch=master)
[![Build Status](https://travis-ci.org/kostiamol/fridgems.svg?branch=master)](https://travis-ci.org/kostiamol/fridgems)
# device-smart-house
Standard dial-up settings.
Sends to: 
HOST = "localhost"
PORT = "3030"
TYPE = "tcp"

Listen to: 
HOST = "localhost"
PORT = "8080"
TYPE = "tcp"

Standard device's config:
It generates random data ever second. It collects and sends data to centre every 5 sec.
Device is turned on from the beginning. 

Used 3rd libraries: 
github.com/Sirupsen/logrus - for logging
