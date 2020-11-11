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

package e2e

import (
	"log"
	"testing"
	"time"

	"github.com/nlopes/slack"

	"github.com/infracloudio/botkube/pkg/bot"
	"github.com/infracloudio/botkube/pkg/controller"
	"github.com/infracloudio/botkube/pkg/notify"
	"github.com/infracloudio/botkube/pkg/utils"
	"github.com/infracloudio/botkube/test/e2e/command"
	"github.com/infracloudio/botkube/test/e2e/env"
	"github.com/infracloudio/botkube/test/e2e/filters"
	"github.com/infracloudio/botkube/test/e2e/notifier/create"
	"github.com/infracloudio/botkube/test/e2e/notifier/delete"
	"github.com/infracloudio/botkube/test/e2e/notifier/error"
	"github.com/infracloudio/botkube/test/e2e/notifier/update"
	"github.com/infracloudio/botkube/test/e2e/welcome"
)

// TestRun run e2e integration tests
func TestRun(t *testing.T) {
	// New Environment to run integration tests
	testEnv := env.New()

	// Fake notifiers
	notifiers := []notify.Notifier{}

	if testEnv.Config.Communications.Slack.Enabled {
		fakeSlackNotifier := &notify.Slack{
			Channel:        testEnv.Config.Communications.Slack.Channel,
			NotifType:      testEnv.Config.Communications.Slack.NotifType,
			AccessBindings: testEnv.Config.Communications.Slack.AccessBindings,
			Client:         slack.New(testEnv.Config.Communications.Slack.Token, slack.OptionAPIURL(testEnv.SlackServer.GetAPIURL())),
		}

		notifiers = append(notifiers, fakeSlackNotifier)
	}

	if testEnv.Config.Communications.Webhook.Enabled {
		fakeWebhookNotifier := &notify.Webhook{
			URL: testEnv.WebhookServer.GetAPIURL(),
		}
		notifiers = append(notifiers, fakeWebhookNotifier)
	}

	utils.DynamicKubeClient = testEnv.K8sClient
	utils.DiscoveryClient = testEnv.DiscoFake
	utils.Mapper = testEnv.Mapper
	utils.InitInformerMap(testEnv.Config)
	utils.InitResourceMap(testEnv.Config)

	// Start controller with fake notifiers
	go controller.RegisterInformers(testEnv.Config, notifiers)
	t.Run("Welcome", welcome.E2ETests(testEnv))

	if testEnv.Config.Communications.Slack.Enabled {
		// Start fake Slack bot
		StartFakeSlackBot(testEnv)
	}

	time.Sleep(time.Second)

	// Make test suite
	suite := map[string]env.E2ETest{
		"notifier": create.E2ETests(testEnv),
		"command":  command.E2ETests(testEnv),
		"filters":  filters.E2ETests(testEnv),
		"update":   update.E2ETests(testEnv),
		"delete":   delete.E2ETests(testEnv),
		"error":    error.E2ETests(testEnv),
	}

	// Run test suite
	for name, test := range suite {
		t.Run(name, test.Run)
	}
}

// StartFakeSlackBot makes connection to mocked slack apiserver
func StartFakeSlackBot(testenv *env.TestEnv) {
	if testenv.Config.Communications.Slack.Enabled {
		log.Println("Starting fake Slack bot")

		// Fake bot
		sb := &bot.SlackBot{
			Token:            testenv.Config.Communications.Slack.Token,
			AllowKubectl:     testenv.Config.Settings.Kubectl.Enabled,
			RestrictAccess:   testenv.Config.Settings.Kubectl.RestrictAccess,
			ClusterName:      testenv.Config.Settings.ClusterName,
			AccessBindings:   testenv.Config.Communications.Slack.AccessBindings,
			SlackURL:         testenv.SlackServer.GetAPIURL(),
			BotID:            testenv.SlackServer.BotID,
			DefaultNamespace: testenv.Config.Settings.Kubectl.DefaultNamespace,
		}
		go sb.Start()
	}
}
