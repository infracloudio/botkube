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
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/events"
	"github.com/infracloudio/botkube/pkg/log"
)

// Webhook contains URL
type Webhook struct {
	URL string
}

// WebhookPayload contains json payload to be sent to webhook url
type WebhookPayload struct {
	EventMeta       EventMeta   `json:"meta"`
	EventStatus     EventStatus `json:"status"`
	EventSummary    string      `json:"summary"`
	TimeStamp       time.Time   `json:"timestamp"`
	Recommendations []string    `json:"recommendations,omitempty"`
	Warnings        []string    `json:"warnings,omitempty"`
}

// EventMeta contains the meta data about the event occurred
type EventMeta struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Cluster   string `json:"cluster,omitempty"`
}

// EventStatus contains the status about the event occurred
type EventStatus struct {
	Type     config.EventType `json:"type"`
	Level    config.Level     `json:"level"`
	Reason   string           `json:"reason,omitempty"`
	Error    string           `json:"error,omitempty"`
	Messages []string         `json:"messages,omitempty"`
}

// NewWebhook returns new Webhook object
func NewWebhook(c config.CommunicationsConfig) Notifier {
	return &Webhook{
		URL: c.Webhook.URL,
	}
}

// SendEvent sends event notification to Webhook url
func (w *Webhook) SendEvent(event events.Event) (err error) {
	jsonPayload := &WebhookPayload{
		EventMeta: EventMeta{
			Kind:      event.Kind,
			Name:      event.Name,
			Namespace: event.Namespace,
			Cluster:   event.Cluster,
		},
		EventStatus: EventStatus{
			Type:     event.Type,
			Level:    event.Level,
			Reason:   event.Reason,
			Error:    event.Error,
			Messages: event.Messages,
		},
		EventSummary:    formatShortMessage(event),
		TimeStamp:       event.TimeStamp,
		Recommendations: event.Recommendations,
		Warnings:        event.Warnings,
	}

	err = w.PostWebhook(jsonPayload)
	if err != nil {
		log.Error(err.Error())
		log.Debugf("Event Not Sent to Webhook %v", event)
	}

	log.Debugf("Event successfully sent to Webhook %v", event)
	return nil
}

// SendMessage sends message to Webhook url
func (w *Webhook) SendMessage(msg string) error {
	return nil
}

// PostWebhook posts webhook to listener
func (w *Webhook) PostWebhook(jsonPayload *WebhookPayload) error {

	message, err := json.Marshal(jsonPayload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", w.URL, bytes.NewBuffer(message))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Error Posting Webhook: %s", string(resp.StatusCode))
	}

	return nil
}
