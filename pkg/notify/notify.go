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
	"fmt"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/events"
	"github.com/infracloudio/botkube/pkg/log"
)

// Notifier to send event notification on the communication channels
type Notifier interface {
	SendEvent(events.Event) error
	SendMessage(string) error
}

func ListNotifiers(conf config.CommunicationsConfig) []Notifier {
	var notifiers []Notifier
	if conf.Slack.Enabled {
		notifiers = append(notifiers, NewSlack(conf.Slack))
	}
	if conf.Mattermost.Enabled {
		if notifier, err := NewMattermost(conf.Mattermost); err == nil {
			notifiers = append(notifiers, notifier)
		} else {
			log.Error(fmt.Sprintf("Failed to create Mattermost client. Error: %v", err))
		}
	}
	if conf.ElasticSearch.Enabled {
		if els, err := NewElasticSearch(conf.ElasticSearch); err == nil {
			notifiers = append(notifiers, els)
		} else {
			log.Error(fmt.Sprintf("Failed to create els client. Error: %v", err))
		}
	}
	if conf.Webhook.Enabled {
		notifiers = append(notifiers, NewWebhook(conf))
	}
	return notifiers
}
