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

package utils

import (
	"context"
	"testing"

	"github.com/nlopes/slack"
	v1 "k8s.io/api/core/v1"
	networkV1beta1 "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/notify"
	"github.com/infracloudio/botkube/pkg/utils"
)

// SlackMessage structure
type SlackMessage struct {
	Text        string
	Attachments []slack.Attachment
}

// WebhookPayload structure
type WebhookPayload struct {
	Summary     string             `json:"summary"`
	EventMeta   notify.EventMeta   `json:"meta"`
	EventStatus notify.EventStatus `json:"status"`
}

// CreateObjects stores specs for creating a k8s fake object and expected Slack response
type CreateObjects struct {
	Kind                   string
	Namespace              string
	Specs                  runtime.Object
	NotifType              config.NotifType
	ExpectedWebhookPayload WebhookPayload
	ExpectedSlackMessage   SlackMessage
}

// CreateResource with fake client
func CreateResource(t *testing.T, obj CreateObjects) {
	switch obj.Kind {
	case "pod":
		s := obj.Specs.(*v1.Pod)
		_, err := utils.KubeClient.CoreV1().Pods(obj.Namespace).Create(context.TODO(), s, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("Failed to create pod: %v", err)
		}
	case "service":
		s := obj.Specs.(*v1.Service)
		_, err := utils.KubeClient.CoreV1().Services(obj.Namespace).Create(context.TODO(), s, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("Failed to create service: %v", err)
		}
	case "ingress":
		s := obj.Specs.(*networkV1beta1.Ingress)
		_, err := utils.KubeClient.NetworkingV1beta1().Ingresses(obj.Namespace).Create(context.TODO(), s, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("Failed to create service: %v", err)
		}
	case "namespace":
		s := obj.Specs.(*v1.Namespace)
		_, err := utils.KubeClient.CoreV1().Namespaces().Create(context.TODO(), s, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("Failed to create service: %v", err)
		}
	default:
		t.Fatalf("CreateResource method is not defined for resource %s", obj.Kind)
	}
}
