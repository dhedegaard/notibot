package main

import (
	"encoding/json"
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
var channelNames map[string]string

func init() {
	usersOnline = make(map[string]struct{})
	channelNames = make(map[string]string)
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
	for _, channel := range guild.Channels {
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
	if len(os.Args) != 3 {
		panic("Please start the application with email and password as parameters.")
	}
	username := os.Args[1]
	password := os.Args[2]
	startTime := time.Now()
	logInfo("Logging in...")
	session, err := discordgo.New(username, password)
	if err != nil {
		panic(err)
	}
	logInfo("Opening session...")
	err = session.Open()
	if err != nil {
		panic(err)
	}
	guilds, err := session.UserGuilds()
	if err != nil {
		panic(err)
	}
	if len(guilds) == 0 {
		panic("No guilds on the user.")
	}
	firstGuild := guilds[0]
	logInfo("Fetching guild state:", firstGuild.Name)
	firstGuild, err = session.Guild(firstGuild.ID)
	if err != nil {
		panic(err)
	}
	logInfo("Fetching presenses...")
	// Setup initial state, ie online users and games played.
	for _, presence := range firstGuild.Presences {
		u := fetchUser(session, presence.User.ID)
		usersOnline[u.Username] = struct{}{}
		logInfo("  User online:", u.Username)
	}
	// Add initial states for channel names.
	logInfo("Fetching channels...")
	for _, channel := range firstGuild.Channels {
		channel, err = session.Channel(channel.ID)
		if err != nil {
			panic(err)
		}
		channelNames[channel.ID] = channel.Name
		logInfo("  Channel added:", channel.Name, "- type:", channel.Type)
	}
	logInfo("Added online/idle users:", len(usersOnline))

	logInfo("Setting up event handlers...")
	session.OnEvent = func(sess *discordgo.Session, evt *discordgo.Event) {
		if evt.Type == "MESSAGE_CREATE" {
			message := messageType{}
			err = json.Unmarshal(evt.RawData, &message)
			if err != nil {
				panic(err)
			}
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
					int(duration.Minutes()),
					int(duration.Seconds()),
					startTime.Format(time.Stamp),
					hostname))
			default:
				// Handle mentions
				for _, elem := range message.Mentions {
					switch elem := elem.(type) {
					case map[string]interface{}:
						if str, ok := elem["username"].(string); ok && strings.ToLower(str) == "notibot" {
							u := fetchUser(sess, message.Author.ID)
							sendMessage(sess, fmt.Sprintf("Hi **%s** !!", u.Username))
						}
					default:
						logDebug(fmt.Sprintf("type: %T", elem))
					}
				}
			}
		} else {
			logDebug("EVENT:", evt)
		}
	}

	session.OnUserUpdate = func(ses *discordgo.Session, evt *discordgo.User) {
		logDebug("USER UPDATE:", evt)
	}

	session.OnUserSettingsUpdate = func(sess *discordgo.Session, data map[string]interface{}) {
		logDebug("USER SETTINGS UPDATE:", data)
	}

	session.OnVoiceStateUpdate = func(sess *discordgo.Session, evt *discordgo.VoiceState) {
		logDebug("VOICE STATE:", evt)
	}

	session.OnChannelCreate = func(sess *discordgo.Session, channel *discordgo.Channel) {
		logDebug("CHANNEL CREATE:", channel)
		channelNames[channel.ID] = channel.Name
		sendMessage(sess, fmt.Sprintf("Channel created **%s**", channel.Name))
	}

	session.OnChannelDelete = func(sess *discordgo.Session, channel *discordgo.Channel) {
		logDebug("CHANNEL DELETED:", channel.Name)
		delete(channelNames, channel.ID)
		sendMessage(sess, fmt.Sprintf("Channel deleted **%s**", channel.Name))
	}

	session.OnChannelUpdate = func(sess *discordgo.Session, channel *discordgo.Channel) {
		logDebug("CHANNEL UPDATE:", channel)
		oldChannel, ok := channelNames[channel.ID]
		if !ok || oldChannel != channel.Name {
			sendMessage(sess, fmt.Sprintf(
				"Channel name changed from **%s** to **%s**", oldChannel, channel.Name))
			channelNames[channel.ID] = channel.Name
		}
	}

	session.OnPresenceUpdate = func(sess *discordgo.Session, evt *discordgo.PresenceUpdate) {
		logDebug("PRESENSE UPDATE:", evt)
		u := fetchUser(sess, evt.User.ID)
		// Ignore self
		if u.Email == username {
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
	}

	logInfo("Sleeping...")
	select {}
}
