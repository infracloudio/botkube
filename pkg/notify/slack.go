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
	"github.com/nlopes/slack"
)

var attachmentColor = map[config.Level]string{
	config.Info:     "good",
	config.Warn:     "warning",
	config.Debug:    "good",
	config.Error:    "danger",
	config.Critical: "danger",
}

// Slack contains Token for authentication with slack and Channel name to send notification to
type Slack struct {
	Channel   string
	NotifType config.NotifType
	Client    *slack.Client
}

// NewSlack returns new Slack object
func NewSlack(c config.Slack) Notifier {
	return &Slack{
		Channel:   c.Channel,
		NotifType: c.NotifType,
		Client:    slack.New(c.Token),
	}
}

// SendEvent sends event notification to slack
func (s *Slack) SendEvent(event events.Event) error {
	log.Debug(fmt.Sprintf(">> Sending to slack: %+v", event))
	attachment := formatSlackMessage(event, s.NotifType)

	// non empty value in event.channel demands redirection of events to a different channel
	if event.Channel != "" {
		channelID, timestamp, err := s.Client.PostMessage(event.Channel, slack.MsgOptionAttachments(attachment), slack.MsgOptionAsUser(true))
		if err != nil {
			log.Errorf("Error in sending slack message %s", err.Error())
			// send error message to default channel
			if err.Error() == "channel_not_found" {
				msg := fmt.Sprintf("Unable to send message to Channel `%s`: `%s`\n```add Botkube app to the Channel %s\nMissed events follows below:```", event.Channel, err.Error(), event.Channel)
				go s.SendMessage(msg)
				// sending missed event to default channel
				// reset event.Channel and send event
				event.Channel = ""
				go s.SendEvent(event)
			}
			return err
		}
		log.Debugf("Event successfully sent to channel %s at %s", channelID, timestamp)
	} else {
		// empty value in event.channel sends notifications to default channel.
		channelID, timestamp, err := s.Client.PostMessage(s.Channel, slack.MsgOptionAttachments(attachment), slack.MsgOptionAsUser(true))
		if err != nil {
			log.Errorf("Error in sending slack message %s", err.Error())
			return err
		}
		log.Debugf("Event successfully sent to channel %s at %s", channelID, timestamp)
	}
	return nil
}

// SendMessage sends message to slack channel
func (s *Slack) SendMessage(msg string) error {
	log.Debug(fmt.Sprintf(">> Sending to slack: %+v", msg))

	//api := slack.New(s.Token)
	//if len(s.SlackURL) != 0 {
	//	api = slack.New(s.Token, slack.OptionAPIURL(s.SlackURL))
	//}

	channelID, timestamp, err := s.Client.PostMessage(s.Channel, slack.MsgOptionText(msg, false), slack.MsgOptionAsUser(true))
	if err != nil {
		log.Errorf("Error in sending slack message %s", err.Error())
		return err
	}

	log.Debugf("Message successfully sent to channel %s at %s", channelID, timestamp)
	return nil
}

func formatSlackMessage(event events.Event, notifyType config.NotifType) (attachment slack.Attachment) {
	switch notifyType {
	case config.LongNotify:
		attachment = slackLongNotification(event)

	case config.ShortNotify:
		fallthrough

	default:
		// set missing cluster name to event object
		attachment = slackShortNotification(event)
	}

	// Add timestamp
	ts := json.Number(strconv.FormatInt(event.TimeStamp.Unix(), 10))
	if ts > "0" {
		attachment.Ts = ts
	}
	attachment.Color = attachmentColor[event.Level]
	return attachment
}

func slackLongNotification(event events.Event) slack.Attachment {
	attachment := slack.Attachment{
		Pretext: fmt.Sprintf("*%s*", event.Title),
		Fields: []slack.AttachmentField{
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
		},
		Footer: "BotKube",
	}
	if event.Namespace != "" {
		attachment.Fields = append(attachment.Fields, slack.AttachmentField{
			Title: "Namespace",
			Value: event.Namespace,
			Short: true,
		})
	}

	if event.Reason != "" {
		attachment.Fields = append(attachment.Fields, slack.AttachmentField{
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
		attachment.Fields = append(attachment.Fields, slack.AttachmentField{
			Title: "Message",
			Value: message,
		})
	}

	if event.Action != "" {
		attachment.Fields = append(attachment.Fields, slack.AttachmentField{
			Title: "Action",
			Value: event.Action,
		})
	}

	if len(event.Recommendations) > 0 {
		rec := ""
		for _, r := range event.Recommendations {
			rec += fmt.Sprintf("%s\n", r)
		}
		attachment.Fields = append(attachment.Fields, slack.AttachmentField{
			Title: "Recommendations",
			Value: rec,
		})
	}

	if len(event.Warnings) > 0 {
		warn := ""
		for _, w := range event.Warnings {
			warn += fmt.Sprintf("%s\n", w)
		}
		attachment.Fields = append(attachment.Fields, slack.AttachmentField{
			Title: "Warnings",
			Value: warn,
		})
	}

	// Add clustername in the message
	attachment.Fields = append(attachment.Fields, slack.AttachmentField{
		Title: "Cluster",
		Value: event.Cluster,
	})
	return attachment
}

func slackShortNotification(event events.Event) slack.Attachment {
	return slack.Attachment{
		Title: event.Title,
		Fields: []slack.AttachmentField{
			{
				Value: formatShortMessage(event),
			},
		},
		Footer: "BotKube",
	}
}

func formatShortMessage(event events.Event) (msg string) {
	additionalMsg := ""
	if len(event.Messages) > 0 {
		for _, m := range event.Messages {
			additionalMsg += fmt.Sprintf("%s\n", m)
		}
	}
	if len(event.Recommendations) > 0 {
		recommend := ""
		for _, m := range event.Recommendations {
			recommend += fmt.Sprintf("- %s\n", m)
		}
		additionalMsg += fmt.Sprintf("Recommendations:\n%s", recommend)
	}
	if len(event.Warnings) > 0 {
		warning := ""
		for _, m := range event.Warnings {
			warning += fmt.Sprintf("- %s\n", m)
		}
		additionalMsg += fmt.Sprintf("Warnings:\n%s", warning)
	}

	switch event.Type {
	case config.CreateEvent, config.DeleteEvent, config.UpdateEvent:
		switch event.Kind {
		case "Namespace", "Node", "PersistentVolume", "ClusterRole", "ClusterRoleBinding":
			msg = fmt.Sprintf(
				"%s *%s* has been %s in *%s* cluster\n",
				event.Kind,
				event.Name,
				event.Type+"d",
				event.Cluster,
			)
		default:
			msg = fmt.Sprintf(
				"%s *%s/%s* has been %s in *%s* cluster\n",
				event.Kind,
				event.Namespace,
				event.Name,
				event.Type+"d",
				event.Cluster,
			)
		}
	case config.ErrorEvent:
		switch event.Kind {
		case "Namespace", "Node", "PersistentVolume", "ClusterRole", "ClusterRoleBinding":
			msg = fmt.Sprintf(
				"Error Occurred in %s: *%s* in *%s* cluster\n",
				event.Kind,
				event.Name,
				event.Cluster,
			)
		default:
			msg = fmt.Sprintf(
				"Error Occurred in %s: *%s/%s* in *%s* cluster\n",
				event.Kind,
				event.Namespace,
				event.Name,
				event.Cluster,
			)
		}
	case config.WarningEvent:
		switch event.Kind {
		case "Namespace", "Node", "PersistentVolume", "ClusterRole", "ClusterRoleBinding":
			msg = fmt.Sprintf(
				"Warning %s: *%s* in *%s* cluster\n",
				event.Kind,
				event.Name,
				event.Cluster,
			)
		default:
			msg = fmt.Sprintf(
				"Warning %s: *%s/%s* in *%s* cluster\n",
				event.Kind,
				event.Namespace,
				event.Name,
				event.Cluster,
			)
		}
	case config.InfoEvent, config.NormalEvent:
		switch event.Kind {
		case "Namespace", "Node", "PersistentVolume", "ClusterRole", "ClusterRoleBinding":
			msg = fmt.Sprintf(
				"%s Info: *%s* in *%s* cluster\n",
				event.Kind,
				event.Name,
				event.Cluster,
			)
		default:
			msg = fmt.Sprintf(
				"%s Info: *%s/%s* in *%s* cluster\n",
				event.Kind,
				event.Namespace,
				event.Name,
				event.Cluster,
			)
		}
	case config.AuditEvent:
		msg = fmt.Sprintf("%s\n", event.Description)
	}

	// Add message in the attachment if there is any
	if len(additionalMsg) > 0 {
		msg += fmt.Sprintf("```\n%s```", additionalMsg)
	}
	return msg
}
