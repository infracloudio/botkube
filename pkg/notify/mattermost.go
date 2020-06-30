// Copyright (c) 2019 InfraCloud Technologies
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
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/events"
	"github.com/infracloudio/botkube/pkg/log"
	"github.com/mattermost/mattermost-server/model"
)

// Mattermost contains server URL and token
type Mattermost struct {
	Client    *model.Client4
	Channel   string
	NotifType config.NotifType
}

// NewMattermost returns new Mattermost object
func NewMattermost(c config.Mattermost) (Notifier, error) {
	// Set configurations for Mattermost server
	client := model.NewAPIv4Client(c.URL)
	client.SetOAuthToken(c.Token)
	botTeam, resp := client.GetTeamByName(c.Team, "")
	if resp.Error != nil {
		return nil, resp.Error
	}
	botChannel, resp := client.GetChannelByName(c.Channel, botTeam.Id, "")
	if resp.Error != nil {
		return nil, resp.Error
	}

	return &Mattermost{
		Client:    client,
		Channel:   botChannel.Id,
		NotifType: c.NotifType,
	}, nil
}

// SendEvent sends event notification to Mattermost
func (m *Mattermost) SendEvent(event events.Event) error {
	log.Info(fmt.Sprintf(">> Sending to Mattermost: %+v", event))

	var fields []*model.SlackAttachmentField

	switch m.NotifType {
	case config.LongNotify:
		fields = mmLongNotification(event)
	case config.ShortNotify:
		fallthrough

	default:
		// set missing cluster name to event object
		fields = mmShortNotification(event)
	}

	attachment := []*model.SlackAttachment{
		{
			Color:     attachmentColor[event.Level],
			Title:     event.Title,
			Fields:    fields,
			Footer:    "BotKube",
			Timestamp: json.Number(strconv.FormatInt(event.TimeStamp.Unix(), 10)),
		},
	}

	post := &model.Post{}
	post.Props = map[string]interface{}{
		"attachments": attachment,
	}

	// non empty value in event.channel demands redirection of events to a different channel
	if event.Channel != "" {
		post.ChannelId = event.Channel

		if _, resp := m.Client.CreatePost(post); resp.Error != nil {
			log.Error("Failed to send message. Error: ", resp.Error)
			// send error message to default channel
			msg := fmt.Sprintf("Unable to send message to Channel `%s`: `%s`\n```add Botkube app to the Channel %s\nMissed events follows below:```", event.Channel, resp.Error, event.Channel)
			go m.SendMessage(msg)
			// sending missed event to default channel
			// reset event.Channel and send event
			event.Channel = ""
			go m.SendEvent(event)
			return resp.Error
		}
		log.Debugf("Event successfully sent to channel %s", post.ChannelId)
	} else {
		post.ChannelId = m.Channel
		// empty value in event.channel sends notifications to default channel.
		if _, resp := m.Client.CreatePost(post); resp.Error != nil {
			log.Error("Failed to send message. Error: ", resp.Error)
			return resp.Error
		}
		log.Debugf("Event successfully sent to channel %s", post.ChannelId)
	}
	return nil
}

// SendMessage sends message to Mattermost channel
func (m *Mattermost) SendMessage(msg string) error {
	post := &model.Post{}
	post.ChannelId = m.Channel
	post.Message = msg
	if _, resp := m.Client.CreatePost(post); resp.Error != nil {
		log.Error("Failed to send message. Error: ", resp.Error)
	}
	return nil
}

func mmLongNotification(event events.Event) []*model.SlackAttachmentField {
	fields := []*model.SlackAttachmentField{
		{
			Title: "Kind",
			Value: event.Kind,
			Short: true,
		},
		{
			Title: "Name",
			Value: event.Name,
			Short: true,
		},
	}

	if event.Namespace != "" {
		fields = append(fields, &model.SlackAttachmentField{
			Title: "Namespace",
			Value: event.Namespace,
			Short: true,
		})
	}

	if event.Reason != "" {
		fields = append(fields, &model.SlackAttachmentField{
			Title: "Reason",
			Value: event.Reason,
			Short: true,
		})
	}

	if len(event.Messages) > 0 {
		message := ""
		for _, m := range event.Messages {
			message += fmt.Sprintf("%s\n", m)
		}
		fields = append(fields, &model.SlackAttachmentField{
			Title: "Message",
			Value: message,
		})
	}

	if event.Action != "" {
		fields = append(fields, &model.SlackAttachmentField{
			Title: "Action",
			Value: event.Action,
		})
	}

	if len(event.Recommendations) > 0 {
		rec := ""
		for _, r := range event.Recommendations {
			rec += fmt.Sprintf("%s\n", r)
		}
		fields = append(fields, &model.SlackAttachmentField{
			Title: "Recommendations",
			Value: rec,
		})
	}

	if len(event.Warnings) > 0 {
		warn := ""
		for _, w := range event.Warnings {
			warn += fmt.Sprintf("%s\n", w)
		}

		fields = append(fields, &model.SlackAttachmentField{
			Title: "Warnings",
			Value: warn,
		})
	}

	// Add clustername in the message
	fields = append(fields, &model.SlackAttachmentField{
		Title: "Cluster",
		Value: event.Cluster,
	})
	return fields
}

func mmShortNotification(event events.Event) []*model.SlackAttachmentField {
	return []*model.SlackAttachmentField{
		{
			Value: formatShortMessage(event),
		},
	}
}
