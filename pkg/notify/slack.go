package notify

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/events"
	log "github.com/infracloudio/botkube/pkg/logging"
	"github.com/nlopes/slack"
)

var attachmentColor = map[events.Level]string{
	events.Info:     "good",
	events.Warn:     "warning",
	events.Debug:    "good",
	events.Error:    "danger",
	events.Critical: "danger",
}

// Slack contains Token for authentication with slack and Channel name to send notification to
type Slack struct {
	Token     string
	Channel   string
	NotifType config.NotifType
	SlackURL  string // Useful only for testing
}

// NewSlack returns new Slack object
func NewSlack(c *config.Config) Notifier {
	return &Slack{
		Token:     c.Communications.Slack.Token,
		Channel:   c.Communications.Slack.Channel,
		NotifType: c.Communications.Slack.NotifType,
	}
}

// SendEvent sends event notification to slack
func (s *Slack) SendEvent(event events.Event) error {
	log.Logger.Debug(fmt.Sprintf(">> Sending to slack: %+v", event))

	api := slack.New(s.Token)
	if len(s.SlackURL) != 0 {
		api = slack.New(s.Token, slack.OptionAPIURL(s.SlackURL))
	}
	attachment := formatSlackMessage(event, s.NotifType)

	// non empty value in event.channel demands redirection of events to a different channel
	if event.Channel != "" {
		channelID, timestamp, err := api.PostMessage(event.Channel, slack.MsgOptionAttachments(attachment), slack.MsgOptionAsUser(true))
		if err != nil {
			log.Logger.Errorf("Error in sending slack message %s", err.Error())
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
		log.Logger.Debugf("Event successfully sent to channel %s at %s", channelID, timestamp)
	} else {
		// empty value in event.channel sends notifications to default channel.
		channelID, timestamp, err := api.PostMessage(s.Channel, slack.MsgOptionAttachments(attachment), slack.MsgOptionAsUser(true))
		if err != nil {
			log.Logger.Errorf("Error in sending slack message %s", err.Error())
			return err
		}
		log.Logger.Debugf("Event successfully sent to channel %s at %s", channelID, timestamp)
	}
	return nil
}

// SendMessage sends message to slack channel
func (s *Slack) SendMessage(msg string) error {
	log.Logger.Debug(fmt.Sprintf(">> Sending to slack: %+v", msg))

	api := slack.New(s.Token)
	if len(s.SlackURL) != 0 {
		api = slack.New(s.Token, slack.OptionAPIURL(s.SlackURL))
	}

	channelID, timestamp, err := api.PostMessage(s.Channel, slack.MsgOptionText(msg, false), slack.MsgOptionAsUser(true))
	if err != nil {
		log.Logger.Errorf("Error in sending slack message %s", err.Error())
		return err
	}

	log.Logger.Debugf("Message successfully sent to channel %s at %s", channelID, timestamp)
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
	}

	// Add message in the attachment if there is any
	if len(additionalMsg) > 0 {
		msg += fmt.Sprintf("```\n%s```", additionalMsg)
	}
	return msg
}
