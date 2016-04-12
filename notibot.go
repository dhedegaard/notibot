package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

var logger *log.Logger
var usersOnline map[string]struct{}
var startTime time.Time

func init() {
	usersOnline = make(map[string]struct{})
	logger = log.New(os.Stderr, "  ", log.Ltime)
	startTime = time.Now()
}

func logDebug(v ...interface{}) {
	logger.SetPrefix("DEBUG ")
	logger.Println(v...)
}

func logInfo(v ...interface{}) {
	logger.SetPrefix("INFO  ")
	logger.Println(v...)
}

func fetchUser(sess *discordgo.Session, userid string) *discordgo.User {
	result, err := sess.User(userid)
	if err != nil {
		panic(err)
	}
	return result
}

func fetchPrimaryTextChannel(sess *discordgo.Session) *discordgo.Channel {
	guilds, err := sess.UserGuilds()
	if err != nil {
		panic(err)
	}
	guild, err := sess.Guild(guilds[0].ID)
	if err != nil {
		panic(err)
	}
	channels, err := sess.GuildChannels(guild.ID)
	if err != nil {
		panic(err)
	}
	for _, channel := range channels {
		channel, err = sess.Channel(channel.ID)
		if err != nil {
			panic(err)
		}
		if channel.Type == "text" {
			return channel
		}
	}
	return nil
}

func sendMessage(sess *discordgo.Session, message string) {
	channel := fetchPrimaryTextChannel(sess)
	if channel == nil {
		logInfo("Unable to fetch default channel")
		return
	}
	logInfo("SENDING MESSAGE:", message)
	sess.ChannelMessageSend(channel.ID, message)
}

func main() {
	argCount := len(os.Args)
	if argCount < 2 || argCount > 3 {
		panic(errors.New(
			"Please start the application with <email> <password> " +
				"or <app bot user token> as parameter(s)."))
	}
	logInfo("Logging in...")
	session, err := discordgo.New(os.Args[1:])
	session.ShouldReconnectOnError = true
	setupHandlers(session)
	if err != nil {
		panic(err)
	}
	logInfo("Opening session...")
	err = session.Open()
	if err != nil {
		panic(err)
	}

	logInfo("Sleeping...")
	select {}
}

func setupHandlers(session *discordgo.Session) {
	logInfo("Setting up event handlers...")

	session.AddHandler(func(sess *discordgo.Session, evt *discordgo.MessageCreate) {
		message := evt.Message
		switch strings.ToLower(strings.TrimSpace(message.Content)) {
		case "!uptime":
			hostname, err := os.Hostname()
			if err != nil {
				panic(err)
			}
			duration := time.Now().Sub(startTime)
			sendMessage(sess, fmt.Sprintf(
				"Uptime is: **%02d:%02d:%02d** (since **%s**) on **%s**",
				int(duration.Hours()),
				int(duration.Minutes())%60,
				int(duration.Seconds())%60,
				startTime.Format(time.Stamp),
				hostname))
		}
	})

	session.AddHandler(func(sess *discordgo.Session, evt *discordgo.PresenceUpdate) {
		logDebug("PRESENSE UPDATE fired for user-ID:", evt.User.ID)
		self := fetchUser(sess, "@me")
		u := fetchUser(sess, evt.User.ID)
		// Ignore self
		if u.ID == self.ID {
			return
		}
		// Handle online/offline notifications
		if evt.Status == "offline" {
			if _, ok := usersOnline[u.ID]; ok {
				delete(usersOnline, u.ID)
				sendMessage(sess, fmt.Sprintf(`**%s** went offline`, u.Username))
			}
		} else {
			if _, ok := usersOnline[u.ID]; !ok {
				usersOnline[u.ID] = struct{}{}
				sendMessage(sess, fmt.Sprintf(`**%s** is now online`, u.Username))
			}
		}
	})

	session.AddHandler(func(sess *discordgo.Session, evt *discordgo.GuildCreate) {
		logInfo("GUILD_CREATE event fired")
		for _, presence := range evt.Presences {
			user := presence.User
			logInfo("Marked user-ID online:", user.ID)
			usersOnline[user.ID] = struct{}{}
		}
	})
}
