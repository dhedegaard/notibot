# Notibot, a simple Discord bot

[![Build Status](https://travis-ci.org/dhedegaard/notibot.svg?branch=master)](https://travis-ci.org/dhedegaard/notibot)

It sends messages to a text channel whenever someone is leaving/joining the
server or creating/changing/deleting channels.

## Running standalone ##

Install dependency

`$ go get github.com/bwmarrin/discordgo`

Run the program

`$ go run notibot.go [email] [password]`

## Running through Docker ##

Pull the image from the [Docker Hub](https://hub.docker.com/r/dhedegaard/notibot/):

`$ docker pull dhedegaard/notibot`

Run the image

`$ docker run -d --name notibot dhedegaard/notibot app [email] [password]`
