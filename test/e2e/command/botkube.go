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

package command

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/nlopes/slack"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/execute"
	"github.com/infracloudio/botkube/pkg/utils"
	testutils "github.com/infracloudio/botkube/test/e2e/utils"
)

type botkubeCommand struct {
	command  string
	expected string
}

// Send botkube command via Slack message and check if BotKube returns correct response
func (c *context) testBotkubeCommand(t *testing.T) {
	botkubeVersion := os.Getenv("BOTKUBE_VERSION")
	// Test cases
	tests := map[string]botkubeCommand{
		"BotKube ping": {
			command:  "ping",
			expected: fmt.Sprintf("```pong from cluster '%s'\n\nK8s Server Version: %s\nBotKube version: %s```", c.Config.Settings.ClusterName, execute.K8sVersion, botkubeVersion),
		},
		"BotKube filters list": {
			command: "filters list",
			expected: "FILTER                  ENABLED DESCRIPTION\n" +
				"NamespaceChecker        true    Checks if event belongs to blocklisted namespaces and filter them.\n" +
				"NodeEventsChecker       true    Sends notifications on node level critical events.\n" +
				"ObjectAnnotationChecker true    Checks if annotations botkube.io/* present in object specs and filters them.\n" +
				"PodLabelChecker         true    Checks and adds recommendations if labels are missing in the pod specs.\n" +
				"ImageTagChecker         true    Checks and adds recommendation if 'latest' image tag is used for container image.\n" +
				"IngressValidator        true    Checks if services and tls secrets used in ingress specs are available.\n",
		},
		"BotKube commands list": {
			command: "commands list",
			expected: "allowed verbs:\n" +
				"  - api-resources\n" +
				"  - describe\n" +
				"  - diff\n" +
				"  - explain\n" +
				"  - get\n" +
				"  - logs\n" +
				"  - api-versions\n" +
				"  - cluster-info\n" +
				"  - top\n" +
				"  - auth\n" +
				"allowed resources:\n" +
				"  - nodes\n" +
				"  - deployments\n" +
				"  - pods\n" +
				"  - namespaces\n" +
				"  - daemonsets\n" +
				"  - statefulsets\n" +
				"  - storageclasses\n",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			if c.TestEnv.Config.Communications.Slack.Enabled {
				// Send message to a channel
				for _, accessBinding := range c.TestEnv.Config.Communications.Slack.AccessBindings {

					c.SlackServer.SendMessageToBot(accessBinding.ChannelName, test.command)
					// Get last seen slack message
					lastSeenMsg := c.GetLastSeenSlackMessage(1)

					// Convert text message into Slack message structure
					m := slack.Message{}
					err := json.Unmarshal([]byte(*lastSeenMsg), &m)
					assert.NoError(t, err, "message should decode properly")
					assert.Equal(t, accessBinding.ChannelName, m.Channel)
					switch test.command {
					case "filters list":
						fl := compareFilters(strings.Split(test.expected, "\n"), strings.Split(strings.Trim(m.Text, "```"), "\n"))
						assert.Equal(t, fl, true)
					case "commands list":
						cl := compareFilters(strings.Split(test.expected, "\n"), strings.Split(strings.Trim(m.Text, "```"), "\n"))
						assert.Equal(t, cl, true)
					default:
						assert.Equal(t, test.expected, m.Text)
					}
				}
			}
		})
	}
}

func compareFilters(expected, actual []string) bool {
	if len(expected) != len(actual) {
		return false
	}

	// Compare slices
	for _, a := range actual {
		found := false
		for _, e := range expected {
			if a == e {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// Test disable notification with BotKube notifier command
// - disable notifier with '@BotKube notifier stop'
// - create pod and verify BotKube doesn't send notification
// - enable notifier with '@BotKube notifier start'
func (c *context) testNotifierCommand(t *testing.T) {
	// Disable notifier with @BotKube notifier stop
	t.Run("disable notifier", func(t *testing.T) {
		if c.TestEnv.Config.Communications.Slack.Enabled {
			// Send message to a channel
			for _, accessBinding := range c.TestEnv.Config.Communications.Slack.AccessBindings {
				c.SlackServer.SendMessageToBot(accessBinding.ChannelName, "notifier stop")
				// Get last seen slack message
				lastSeenMsg := c.GetLastSeenSlackMessage(1)

				// Convert text message into Slack message structure
				m := slack.Message{}
				err := json.Unmarshal([]byte(*lastSeenMsg), &m)
				assert.NoError(t, err, "message should decode properly")
				assert.Equal(t, accessBinding.ChannelName, m.Channel)
				assert.Equal(t, fmt.Sprintf("```Sure! I won't send you notifications from cluster '%s' anymore.```", c.Config.Settings.ClusterName), m.Text)
				assert.Equal(t, config.Notify, false)

			}
		}
	})

	// Create pod and verify that BotKube is not sending notifications
	pod := testutils.CreateObjects{
		GVR:       schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
		Kind:      "Pod",
		Namespace: "test",
		Specs:     &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod-notifier"}},
		ExpectedSlackMessage: testutils.SlackMessage{
			Attachments: []slack.Attachment{{Color: "good", Fields: []slack.AttachmentField{{Title: "Pod create", Value: "Pod `test-pod` in of cluster `test-cluster-1`, namespace `test` has been created:\n```Resource created\nRecommendations:\n- pod 'test-pod' creation without labels should be avoided.\n```", Short: false}}, Footer: "BotKube"}},
		},
	}
	t.Run("create resource", func(t *testing.T) {
		// Inject an event into the fake client.
		testutils.CreateResource(t, pod)

		if c.TestEnv.Config.Communications.Slack.Enabled {
			for i := range c.Config.Communications.Slack.AccessBindings {

				// Get last seen slack message
				lastSeenMsg := c.GetLastSeenSlackMessage(i + 1)
				// Convert text message into Slack message structure
				m := slack.Message{}
				err := json.Unmarshal([]byte(*lastSeenMsg), &m)
				assert.NoError(t, err, "message should decode properly")
				assert.NotEqual(t, pod.ExpectedSlackMessage.Attachments, m.Attachments)
				// since same message is sent to all the channels, we are comparing
				// that the each new message received on slack must be configured under AccessBindings,
				// and also  all new messages must be same as expacted one\
				validChannel := utils.Contains(utils.GetAllChannels(&c.TestEnv.Config.Communications.Slack.AccessBindings), m.Channel)
				assert.Equal(t, validChannel, true)
			}
		}
	})

	// Revert and Enable notifier
	t.Run("Enable notifier", func(t *testing.T) {
		if c.TestEnv.Config.Communications.Slack.Enabled {
			for _, accessBinding := range c.TestEnv.Config.Communications.Slack.AccessBindings {
				// Send message to a channel
				c.SlackServer.SendMessageToBot(accessBinding.ChannelName, "notifier start")

				// Get last seen slack message
				lastSeenMsg := c.GetLastSeenSlackMessage(1)

				// Convert text message into Slack message structure
				m := slack.Message{}
				err := json.Unmarshal([]byte(*lastSeenMsg), &m)
				assert.NoError(t, err, "message should decode properly")
				assert.Equal(t, accessBinding.ChannelName, m.Channel)
				assert.Equal(t, fmt.Sprintf("```Brace yourselves, notifications are coming from cluster '%s'.```", c.Config.Settings.ClusterName), m.Text)
				assert.Equal(t, config.Notify, true)
			}
		}
	})
}
