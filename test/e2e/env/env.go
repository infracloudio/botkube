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

package env

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/nlopes/slack"
	"github.com/nlopes/slack/slacktest"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	cacheddiscovery "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/fake"
	kubeFake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/restmapper"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/test/e2e/utils"
	"github.com/infracloudio/botkube/test/webhook"
)

// TestEnv to store objects required for e2e testing
// K8sClient    : Fake K8s client to mock resource creation
// SlackServer  : Fake Slack server
// SlackMessages: Channel to store incoming Slack messages from BotKube
// Config	: BotKube config provided with config.yaml
type TestEnv struct {
	DiscoFake     discovery.DiscoveryInterface
	K8sClient     dynamic.Interface
	SlackServer   *slacktest.Server
	WebhookServer *webhook.Server
	SlackMessages chan (*slack.MessageEvent)
	Config        *config.Config
	Mapper        *restmapper.DeferredDiscoveryRESTMapper
}

// E2ETest interface to run tests
type E2ETest interface {
	Run(*testing.T)
}

// New creates TestEnv and populate required objects
func New() *TestEnv {
	testEnv := &TestEnv{}

	// Loads `/test/config.yaml` for Integration Testing
	conf, err := config.New()
	if err != nil {
		log.Fatal(fmt.Sprintf("Error in loading configuration. Error:%s", err.Error()))
	}
	testEnv.Config = conf

	// Set fake BotKube version
	os.Setenv("BOTKUBE_VERSION", "v9.99.9")

	s := runtime.NewScheme()
	testEnv.K8sClient = fake.NewSimpleDynamicClient(s)
	testEnv.DiscoFake = kubeFake.NewSimpleClientset().Discovery()
	discoCacheClient := cacheddiscovery.NewMemCacheClient(FakeCachedDiscoveryInterface())
	testEnv.Mapper = restmapper.NewDeferredDiscoveryRESTMapper(discoCacheClient)

	if testEnv.Config.Communications.Slack.Enabled {
		testEnv.SlackMessages = make(chan (*slack.MessageEvent), 1)
		testEnv.SetupFakeSlack()
	}
	if testEnv.Config.Communications.Webhook.Enabled {
		testEnv.SetupFakeWebhook()
	}

	return testEnv
}

// SetupFakeSlack create fake Slack server to mock Slack
func (e *TestEnv) SetupFakeSlack() {
	s := slacktest.NewTestServer()
	s.SetBotName("BotKube")
	go s.Start()

	e.SlackServer = s
}

// GetLastSeenSlackMessage return last message received by fake slack server, i=1 for last, i=2 for 2nd last and so on
func (e TestEnv) GetLastSeenSlackMessage(i int) *string {
	time.Sleep(5 * time.Second)
	allSeenMessages := e.SlackServer.GetSeenOutboundMessages()
	if len(allSeenMessages) != 0 && len(allSeenMessages) >= i {
		return &allSeenMessages[len(allSeenMessages)-i]
	}
	return nil
}

// SetupFakeWebhook create fake Slack server to mock Slack
func (e *TestEnv) SetupFakeWebhook() {
	s := webhook.NewTestServer()
	go s.Start()

	e.WebhookServer = s
}

// GetLastReceivedPayload return last message received by fake webhook server
func (e TestEnv) GetLastReceivedPayload() *utils.WebhookPayload {

	time.Sleep(5 * time.Second)

	allSeenMessages := e.WebhookServer.GetReceivedPayloads()
	if len(allSeenMessages) != 0 {
		return &allSeenMessages[len(allSeenMessages)-1]
	}
	return nil
}
