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
	"testing"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/execute"
	"github.com/infracloudio/botkube/pkg/utils"
	"github.com/infracloudio/botkube/test/e2e/env"
	"github.com/nlopes/slack"
	"github.com/stretchr/testify/assert"
)

type kubectlCommand struct {
	command   string
	verb      string
	resource  string
	namespace string
	expected  string
}

type context struct {
	*env.TestEnv
}

const (
	unsupportedCmdMsg = "```Command not supported or you don't have access to run this command. Please run /botkubehelp to see supported commands.```"
)

// Send kubectl command via Slack message and check if BotKube returns correct response
func (c *context) testKubectlCommand(t *testing.T) {
	// Test cases
	tests := map[string]kubectlCommand{
		"BotKube get pods from configured channel": {
			command:   "get pods -n default",
			verb:      "get",
			resource:  "pods",
			namespace: "default", //IF namespace is not specified (like get namespace) by defaulte default namespace from settings of resource_confg is appended by botkube
			expected:  fmt.Sprintf("```Cluster: %s\n%s```", c.Config.Settings.ClusterName, execute.KubectlResponse["get pods"]),
		},
		"BotKube get pods from different namespace": {
			command:   "get pods -n kube-system",
			verb:      "get",
			resource:  "pods",
			namespace: "kube-system", //IF namespace is not specified (like get namespace) by defaulte default namespace from settings of resource_confg is appended by botkube
			expected:  fmt.Sprintf("```Cluster: %s\n%s```", c.Config.Settings.ClusterName, execute.KubectlResponse["get pods -n kube-system"]),
		},
		"BotKube get namespaces": {
			command:   "get namespaces",
			verb:      "get",
			resource:  "namespaces",
			namespace: "default",
			expected:  fmt.Sprintf("```Cluster: %s\n%s```", c.Config.Settings.ClusterName, execute.KubectlResponse[" get namespaces"]),
		},
		"kubectl command on forbidden verb and resource": {
			command:  "config set clusters.test-clustor-1.server https://1.2.3.4",
			expected: "```Command not supported. Please run /botkubehelp to see supported commands.```",
		},
		"kubectl command on forbidden resource": {
			command:   "get endpoints",
			verb:      "get",
			resource:  "endpoints",
			namespace: "default",
			expected:  "```Command not supported. Please run /botkubehelp to see supported commands.```",
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

					if checkAllowedMessageByProfile(test.verb, test.resource, test.namespace, accessBinding.ProfileValue, m.Channel) {
						assert.Equal(t, test.expected, m.Text)
					} else {
						assert.Equal(t, unsupportedCmdMsg, m.Text)
					}

				}

			}
		})
	}
}

// checkAllowedMessageByProfile check if chennal should execute provided command based on access permission defined with matching profile mapped to the channel
func checkAllowedMessageByProfile(verb string, resource string, namespace string, profile config.Profile, channel string) bool {

	if utils.Contains(profile.Namespaces, namespace) && utils.Contains(profile.Kubectl.Commands.Verbs, verb) && utils.Contains(profile.Kubectl.Commands.Resources, resource) {
		return true
	}
	return false
}

// Run tests
func (c *context) Run(t *testing.T) {
	// Run kubectl tests
	t.Run("Test Kubectl command", c.testKubectlCommand)
	t.Run("Test BotKube command", c.testBotkubeCommand)
	t.Run("Test disable notifier", c.testNotifierCommand)
}

// E2ETests runs command execution tests
func E2ETests(testEnv *env.TestEnv) env.E2ETest {
	return &context{
		testEnv,
	}
}
