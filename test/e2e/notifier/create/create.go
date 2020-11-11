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

package create

import (
	"encoding/json"
	"testing"

	"github.com/nlopes/slack"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	samplev1alpha1 "k8s.io/sample-controller/pkg/apis/samplecontroller/v1alpha1"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/notify"
	"github.com/infracloudio/botkube/pkg/utils"
	"github.com/infracloudio/botkube/test/e2e/env"
	testutils "github.com/infracloudio/botkube/test/e2e/utils"
)

type context struct {
	*env.TestEnv
}

// Test if BotKube sends notification when a resource is created
func (c *context) testCreateResource(t *testing.T) {
	// Test cases
	tests := map[string]testutils.CreateObjects{
		"create pod in configured namespace": {
			GVR:       schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
			Kind:      "Pod",
			Namespace: "test",
			Specs:     &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod"}},
			ExpectedSlackMessage: testutils.SlackMessage{
				Attachments: []slack.Attachment{{Color: "good", Title: "v1/pods created", Fields: []slack.AttachmentField{{Value: "Pod *test/test-pod* has been created in *test-cluster-1* cluster\n```\nRecommendations:\n- pod 'test-pod' creation without labels should be avoided.\n```", Short: false}}, Footer: "BotKube"}},
			},
			ExpectedWebhookPayload: testutils.WebhookPayload{
				EventMeta:   notify.EventMeta{Kind: "Pod", Name: "test-pod", Namespace: "test", Cluster: "test-cluster-1"},
				EventStatus: notify.EventStatus{Type: "create", Level: "info", Reason: "", Error: ""},
				Summary:     "Pod *test/test-pod* has been created in *test-cluster-1* cluster\n```\nRecommendations:\n- pod 'test-pod' creation without labels should be avoided.\n```",
			},
		},
		"create service in configured namespace": {
			GVR:       schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"},
			Kind:      "Service",
			Namespace: "test",
			Specs:     &v1.Service{ObjectMeta: metav1.ObjectMeta{Name: "test-service"}},
			ExpectedSlackMessage: testutils.SlackMessage{
				Attachments: []slack.Attachment{{Color: "good", Title: "v1/services created", Fields: []slack.AttachmentField{{Value: "Service *test/test-service* has been created in *test-cluster-1* cluster\n", Short: false}}, Footer: "BotKube"}},
			},
			ExpectedWebhookPayload: testutils.WebhookPayload{
				EventMeta:   notify.EventMeta{Kind: "Service", Name: "test-service", Namespace: "test", Cluster: "test-cluster-1"},
				EventStatus: notify.EventStatus{Type: "create", Level: "info", Reason: "", Error: ""},
				Summary:     "Service *test/test-service* has been created in *test-cluster-1* cluster\n",
			},
		},
		"create a namespace": {
			GVR:   schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"},
			Kind:  "Namespace",
			Specs: &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test-namespace"}},
			ExpectedSlackMessage: testutils.SlackMessage{
				Attachments: []slack.Attachment{{Color: "good", Title: "v1/namespaces created", Fields: []slack.AttachmentField{{Value: "Namespace *test-namespace* has been created in *test-cluster-1* cluster\n", Short: false}}, Footer: "BotKube"}},
			},
			ExpectedWebhookPayload: testutils.WebhookPayload{
				EventMeta:   notify.EventMeta{Kind: "Namespace", Name: "test-namespace", Namespace: "", Cluster: "test-cluster-1"},
				EventStatus: notify.EventStatus{Type: "create", Level: "info", Reason: "", Error: ""},
				Summary:     "Namespace *test-namespace* has been created in *test-cluster-1* cluster\n",
			},
		},
		"create a foo CR": {
			GVR:       schema.GroupVersionResource{Group: "samplecontroller.k8s.io", Version: "v1alpha1", Resource: "foos"},
			Kind:      "Foo",
			Namespace: "test",
			Specs:     &samplev1alpha1.Foo{ObjectMeta: metav1.ObjectMeta{Name: "test-foo"}},
			ExpectedSlackMessage: testutils.SlackMessage{
				Attachments: []slack.Attachment{{Color: "good", Title: "samplecontroller.k8s.io/v1alpha1/foos created", Fields: []slack.AttachmentField{{Value: "Foo *test/test-foo* has been created in *test-cluster-1* cluster\n", Short: false}}, Footer: "BotKube"}},
			},
			ExpectedWebhookPayload: testutils.WebhookPayload{
				EventMeta:   notify.EventMeta{Kind: "Foo", Name: "test-foo", Namespace: "test", Cluster: "test-cluster-1"},
				EventStatus: notify.EventStatus{Type: "create", Level: "info", Reason: "", Error: ""},
				Summary:     "Foo *test/test-foo* has been created in *test-cluster-1* cluster\n",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// Inject an event into the fake client.
			testutils.CreateResource(t, test)

			if c.TestEnv.Config.Communications.Slack.Enabled {

				for i := range c.Config.Communications.Slack.AccessBindings {
					lastSeenMsg := c.GetLastSeenSlackMessage(i + 1)
					// Convert text message into Slack message structure
					m := slack.Message{}
					err := json.Unmarshal([]byte(*lastSeenMsg), &m)
					assert.NoError(t, err, "message should decode properly")
					assert.Equal(t, test.ExpectedSlackMessage.Attachments, m.Attachments)
					// since same message is sent to all the channels, we are comparing
					// that the each new message received on slack must be configured under AccessBindings,
					// and also  all new messages must be same as expacted one\
					validChannel := utils.Contains(utils.GetAllChannels(&c.TestEnv.Config.Communications.Slack.AccessBindings), m.Channel)
					assert.Equal(t, validChannel, true)

				}
			}

			if c.TestEnv.Config.Communications.Webhook.Enabled {
				// Get last seen webhook payload
				lastSeenPayload := c.GetLastReceivedPayload()
				assert.Equal(t, test.ExpectedWebhookPayload.EventMeta, lastSeenPayload.EventMeta)
				assert.Equal(t, test.ExpectedWebhookPayload.EventStatus, lastSeenPayload.EventStatus)
				assert.Equal(t, test.ExpectedWebhookPayload.Summary, lastSeenPayload.Summary)
			}

			resource := utils.GVRToString(test.GVR)
			isAllowed := utils.CheckOperationAllowed(utils.AllowedEventKindsMap, test.Namespace, resource, config.CreateEvent)
			assert.Equal(t, isAllowed, true)
		})
	}
}

// Run tests
func (c *context) Run(t *testing.T) {
	t.Run("create resource", c.testCreateResource)
}

// E2ETests runs create notification tests
func E2ETests(testEnv *env.TestEnv) env.E2ETest {
	return &context{
		testEnv,
	}
}
