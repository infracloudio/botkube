// Copyright (c) 2020 InfraCloud Technologies
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package notify

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/events"
	"github.com/infracloudio/botkube/pkg/log"
)

// customTimeFormat holds custom time format string
const customTimeFormat = "2006-01-02 15:04:05"

var embedColor = map[config.Level]int{
	config.Info:     8311585,  // green
	config.Warn:     16312092, // yellow
	config.Debug:    8311585,  // green
	config.Error:    13632027, // red
	config.Critical: 13632027, // red
}

// Discord contains URL and ClusterName
type Discord struct {
	Token          string
	ChannelID      string
	NotifType      config.NotifType
	AccessBindings []config.AccessBinding
}

// NewDiscord returns new Discord object
func NewDiscord(c config.Discord) Notifier {
	return &Discord{
		Token:          c.Token,
		ChannelID:      c.Channel,
		AccessBindings: c.AccessBindings,
		NotifType:      c.NotifType,
	}
}

// SendEvent sends event notification to Discord Channel
func (d *Discord) SendEvent(event events.Event) (err error) {
	log.Debug(fmt.Sprintf(">> Sending to discord: %+v", event))
	if len(event.DiscordChannels) == 0 {
		// send to all configured channel
		event.DiscordChannels = d.getAllChannels()
	}
	for _, channel := range event.DiscordChannels {
		api, err := discordgo.New("Bot " + d.Token)
		if err != nil {
			log.Error("error creating Discord session,", err)
			return err
		}
		messageSend := formatDiscordMessage(event, d.NotifType)

		if _, err := api.ChannelMessageSendComplex(channel, &messageSend); err != nil {
			log.Errorf("Error in sending message: %+v", err)
			return err
		}
		log.Debugf("Event successfully sent to channel %s", channel)
	}
	return nil
}

// SendMessage sends message to Discord Channel
func (d *Discord) SendMessage(msg string) error {
	log.Debug(fmt.Sprintf(">> Sending to discord: %+v", msg))
	for _, channel := range d.getAllChannels() {
		api, err := discordgo.New("Bot " + d.Token)
		if err != nil {
			log.Error("error creating Discord session,", err)
			return err
		}

		if _, err := api.ChannelMessageSend(channel, msg); err != nil {
			log.Error("Error in sending message:", err)
			return err
		}
		log.Debugf("Event successfully sent to Discord %v", msg)
	}
	return nil
}

func formatDiscordMessage(event events.Event, notifyType config.NotifType) discordgo.MessageSend {

	var messageEmbed discordgo.MessageEmbed

	switch notifyType {
	case config.LongNotify:
		// generate Long notification message
		messageEmbed = discordLongNotification(event)

	case config.ShortNotify:
		// generate Short notification message
		fallthrough

	default:
		// generate Short notification message
		messageEmbed = discordShortNotification(event)
	}

	// Add timestamp
	messageEmbed.Timestamp = event.TimeStamp.Format(customTimeFormat)

	messageEmbed.Color = embedColor[event.Level]

	return discordgo.MessageSend{
		Embed: &messageEmbed,
	}

}

func discordLongNotification(event events.Event) discordgo.MessageEmbed {
	messageEmbed := discordgo.MessageEmbed{
		Title: fmt.Sprintf("*%s*", event.Title),
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Kind",
				Value:  event.Kind,
				Inline: true,
			},
			{

				Name:   "Name",
				Value:  event.Name,
				Inline: true,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "BotKube",
		},
	}
	if event.Namespace != "" {
		messageEmbed.Fields = append(messageEmbed.Fields, &discordgo.MessageEmbedField{
			Name:   "Namespace",
			Value:  event.Namespace,
			Inline: true,
		})
	}

	if event.Reason != "" {
		messageEmbed.Fields = append(messageEmbed.Fields, &discordgo.MessageEmbedField{
			Name:   "Reason",
			Value:  event.Reason,
			Inline: true,
		})
	}

	if len(event.Messages) > 0 {
		message := ""
		for _, m := range event.Messages {
			message += fmt.Sprintf("%s\n", m)
		}
		messageEmbed.Fields = append(messageEmbed.Fields, &discordgo.MessageEmbedField{
			Name:  "Message",
			Value: message,
		})
	}

	if event.Action != "" {
		messageEmbed.Fields = append(messageEmbed.Fields, &discordgo.MessageEmbedField{
			Name:  "Action",
			Value: event.Action,
		})
	}

	if len(event.Recommendations) > 0 {
		rec := ""
		for _, r := range event.Recommendations {
			rec += fmt.Sprintf("%s\n", r)
		}
		messageEmbed.Fields = append(messageEmbed.Fields, &discordgo.MessageEmbedField{
			Name:  "Recommendations",
			Value: rec,
		})
	}

	if len(event.Warnings) > 0 {
		warn := ""
		for _, w := range event.Warnings {
			warn += fmt.Sprintf("%s\n", w)
		}
		messageEmbed.Fields = append(messageEmbed.Fields, &discordgo.MessageEmbedField{
			Name:  "Warnings",
			Value: warn,
		})
	}

	return messageEmbed
}

func discordShortNotification(event events.Event) discordgo.MessageEmbed {
	return discordgo.MessageEmbed{
		Title:       event.Title,
		Description: FormatShortMessage(event),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "BotKube",
		},
	}
}

// getAllChannels return all the channels configured under AccessBinding
func (d *Discord) getAllChannels() []string {
	var allChannels []string
	for _, accessBinding := range d.AccessBindings {
		allChannels = append(allChannels, accessBinding.ChannelName)
	}
	if len(allChannels) == 0 {
		log.Errorf("No channel name found from profiles corresponds to AccessBindings for Slack")
	}
	return allChannels
}
