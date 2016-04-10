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

type messageType struct {
	Attachments []interface{} `json:"attachments"`
	Author      struct {
		Avatar        string `json:"avatar"`
		Discriminator string `json:"discriminator"`
		ID            string `json:"id"`
		Username      string `json:"username"`
	} `json:"author"`
	ChannelID       string        `json:"channel_id"`
	Content         string        `json:"content"`
	EditedTimestamp interface{}   `json:"edited_timestamp"`
	Embeds          []interface{} `json:"embeds"`
	ID              string        `json:"id"`
	MentionEveryone bool          `json:"mention_everyone"`
	Mentions        []interface{} `json:"mentions"`
	Nonce           string        `json:"nonce"`
	Timestamp       string        `json:"timestamp"`
	Tts             bool          `json:"tts"`
}

var logger *log.Logger
var usersOnline map[string]struct{}
var startTime time.Time

func init() {
	usersOnline = make(map[string]struct{})
	logger = log.New(os.Stderr, "  ", log.Ltime)
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
	startTime = time.Now()
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
	self := fetchUser(session, "@me")
	logInfo("This users username is:", self.Username)
	guilds, err := session.UserGuilds()
	if err != nil {
		panic(err)
	}
	if len(guilds) == 0 {
		panic(errors.New("No guilds on the user."))
	}
	firstGuild := guilds[0]
	logInfo("Fetching guild state:", firstGuild.Name)
	firstGuild, err = session.Guild(firstGuild.ID)
	if err != nil {
		panic(err)
	}
	presences := firstGuild.Presences
	logInfo("Fetching presenses...", len(presences))
	// Setup initial state, ie online users and games played.
	for _, presence := range presences {
		u := fetchUser(session, presence.User.ID)
		usersOnline[u.Username] = struct{}{}
		logInfo("  User online:", u.Username)
	}
	logInfo("Added online/idle users:", len(usersOnline))

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
		logDebug("PRESENSE UPDATE:", evt)
		self := fetchUser(sess, "@me")
		u := fetchUser(sess, evt.User.ID)
		// Ignore self
		if u.ID == self.ID {
			return
		}
		// Handle online/offline notifications
		if evt.Status == "offline" {
			if _, ok := usersOnline[u.Username]; ok {
				delete(usersOnline, u.Username)
				sendMessage(sess, fmt.Sprintf(`**%s** went offline`, u.Username))
			}
		} else {
			if _, ok := usersOnline[u.Username]; !ok {
				usersOnline[u.Username] = struct{}{}
				sendMessage(sess, fmt.Sprintf(`**%s** is now online`, u.Username))
			}
		}
	})
}
