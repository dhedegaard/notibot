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
	logger = log.New(os.Stderr, "  ", log.Ldate|log.Ltime)
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

/* Tries to call a method and checking if the method returned an error, if it
did check to see if it's HTTP 502 from the Discord API and retry for
`attempts` number of times. */
func retryOnBadGateway(f func() error) {
	var err error
	for i := 0; i < 3; i++ {
		err = f()
		if err != nil {
			if strings.HasPrefix(err.Error(), "HTTP 502") {
				// If the error is Bad Gateway, try again after 1 sec.
				time.Sleep(1 * time.Second)
				continue
			} else {
				// Otherwise panic !
				panicOnErr(err)
			}
		} else {
			// In case of no error, return.
			return
		}
	}
}

func fetchUser(sess *discordgo.Session, userid string) *discordgo.User {
	var result *discordgo.User
	retryOnBadGateway(func() error {
		var err error
		result, err = sess.User(userid)
		if err != nil {
			return err
		}
		return nil
	})
	return result
}

func fetchPrimaryTextChannelID(sess *discordgo.Session) string {
	var channelid string
	retryOnBadGateway(func() error {
		guilds, err := sess.UserGuilds()
		if err != nil {
			return err
		}
		guild, err := sess.Guild(guilds[0].ID)
		if err != nil {
			return err
		}
		channels, err := sess.GuildChannels(guild.ID)
		if err != nil {
			return err
		}
		for _, channel := range channels {
			channel, err = sess.Channel(channel.ID)
			if err != nil {
				return err
			}
			if channel.Type == "text" {
				channelid = channel.ID
				return nil
			}
		}
		return errors.New("No primary channel found")
	})
	return channelid
}

func sendMessage(sess *discordgo.Session, message string) {
	channelid := fetchPrimaryTextChannelID(sess)
	logInfo("SENDING MESSAGE:", message)
	retryOnBadGateway(func() error {
		_, err := sess.ChannelMessageSend(channelid, message)
		return err
	})
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

	session.AddHandler(func(sess *discordgo.Session, evt *discordgo.Disconnect) {
		logInfo("DISCONNECT event")
	})

	session.AddHandler(func(sess *discordgo.Session, evt *discordgo.Connect) {
		logInfo("CONNECTION event")
	})
}
