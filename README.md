# Notibot, a simple Discord bot
## Running standalone ##

Install dependency

$ go get github.com/bwmarrin/discordgo

Run the program

$ go run notibot.go [email] [password]

## Running through Docker ##

Pull the image from the Docker Hub:

$ docker pull dhedegaard/notibot

Run the image

$ docker run -d --name notibot dhedegaard/notibot app [email] [password]
