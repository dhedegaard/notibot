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

func panicOnErr(err error) {
	if err != nil {
		panic(err)
	}
}

func fetchUser(sess *discordgo.Session, userid string) *discordgo.User {
	result, err := sess.User(userid)
	panicOnErr(err)
	return result
}

func fetchPrimaryTextChannel(sess *discordgo.Session) (*discordgo.Channel, error) {
	guilds, err := sess.UserGuilds()
	if err != nil {
		return nil, err
	}
	guild, err := sess.Guild(guilds[0].ID)
	if err != nil {
		return nil, err
	}
	channels, err := sess.GuildChannels(guild.ID)
	if err != nil {
		return nil, err
	}
	for _, channel := range channels {
		channel, err = sess.Channel(channel.ID)
		if err != nil {
			return nil, err
		}
		if channel.Type == "text" {
			return channel, nil
		}
	}
	return nil, errors.New("No primary channel found")
}

func sendMessage(sess *discordgo.Session, message string) {
	var channelid string
	for i := 0; i < 3; i++ {
		channel, err := fetchPrimaryTextChannel(sess)
		// If an error was returned, handle it.
		if err != nil {
			/* If we get a 502 from the backend, sleep for 1 second and try
			again, except when the error is caught on the third attempt. */
			if i < 2 && strings.HasPrefix(err.Error(), "HTTP 502 Bad Gateway") {
				time.Sleep(1 * time.Second)
				continue
			} else {
				panicOnErr(err)
			}
		}
		// Otherwise a channel was fetched, get the ID and break the loop.
		channelid = channel.ID
		break
	}
	logInfo("SENDING MESSAGE:", message)
	sess.ChannelMessageSend(channelid, message)
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
	panicOnErr(err)
	logInfo("Opening session...")
	err = session.Open()
	panicOnErr(err)

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
			panicOnErr(err)
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
