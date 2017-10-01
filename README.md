# Notibot, a simple Discord bot

[![Build Status](https://travis-ci.org/dhedegaard/notibot.svg?branch=master)](https://travis-ci.org/dhedegaard/notibot)
[![Docker Pulls](https://img.shields.io/docker/pulls/dhedegaard/notibot.svg)](https://hub.docker.com/r/dhedegaard/notibot/)

It sends messages to a text channel whenever someone is leaving/joining the
server.

## Running standalone ##

Install dependencies

`$ go get -v .`

Run the program with a BOT user token

`$ go run notibot.go -t [app bot user token]`

## Running through Docker ##

Pull the image from the [Docker Hub](https://hub.docker.com/r/dhedegaard/notibot/):

`$ docker pull dhedegaard/notibot`

Run the image

`$ docker run -d --name notibot dhedegaard/notibot app -t [app bot user token]`

You might also want to add `--restart always` as parameter for the container
to automatically restart it if/when the process crashes.
