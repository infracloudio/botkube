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

package welcome

import (
	"encoding/json"
	"testing"

	"github.com/infracloudio/botkube/pkg/utils"
	"github.com/infracloudio/botkube/test/e2e/env"
	"github.com/nlopes/slack"
	"github.com/stretchr/testify/assert"
)

type context struct {
	*env.TestEnv
}

// Test if BotKube sends welcome message to the configured slack channel after start
func (c *context) testWelcome(t *testing.T) {
	expected := "...and now my watch begins for cluster 'test-cluster-1'! :crossed_swords:"

	if c.TestEnv.Config.Communications.Slack.Enabled {
		// To validate welcome message was sent on all channels
		for i := range c.TestEnv.Config.Communications.Slack.AccessBindings {
			// Get the last seen message on all configured channels one by one
			lastSeenMsg := c.GetLastSeenSlackMessage(i + 1)
			// Convert text message into Slack message structure
			m := slack.Message{}
			err := json.Unmarshal([]byte(*lastSeenMsg), &m)
			assert.NoError(t, err, "message should decode properly")
			assert.Equal(t, expected, m.Text)
			// since same message is sent to all the channels, we are comparing
			// that the each new message received on slack must be configured under AccessBindings,
			// and also  all new messages must be same as expacted one\
			validChannel := utils.Contains(utils.GetAllChannels(&c.TestEnv.Config.Communications.Slack.AccessBindings), m.Channel)
			assert.Equal(t, validChannel, true)
		}
	}
}

// E2ETests run welcome tests
func E2ETests(testEnv *env.TestEnv) func(*testing.T) {
	ctx := &context{
		testEnv,
	}

	return ctx.testWelcome
}
