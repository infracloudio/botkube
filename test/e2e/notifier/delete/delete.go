package delete

import (
	"encoding/json"
	"testing"

	"github.com/nlopes/slack"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/notify"
	"github.com/infracloudio/botkube/pkg/utils"
	"github.com/infracloudio/botkube/test/e2e/env"
	testutils "github.com/infracloudio/botkube/test/e2e/utils"
)

type context struct {
	*env.TestEnv
}

func (c *context) testSKipDeleteEvent(t *testing.T) {
	// Modifying AllowedEventKindsMap to remove delete event for pod resource
	delete(utils.AllowedEventKindsMap, utils.EventKind{Resource: "v1/pods", Namespace: "all", EventType: "delete"})
	// Modifying AllowedEventKindsMap to add delete event for only dummy namespace and ignore everything
	utils.AllowedEventKindsMap[utils.EventKind{Resource: "v1/pods", Namespace: "dummy", EventType: "delete"}] = true
	// Modifying AllowedEventKindsMap to remove all event for service resource
	events := []config.EventType{"delete", "error", "create"}
	for _, event := range events {
		delete(utils.AllowedEventKindsMap, utils.EventKind{Resource: "v1/services", Namespace: "all", EventType: event})
	}
	// test scenarios
	tests := map[string]testutils.DeleteObjects{
		"skip delete event for resources not configured": {
			// delete operation not allowed for Pod resources so event should be skipped
			GVR:       schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
			Kind:      "Pod",
			Namespace: "test",
			Specs:     &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod-delete"}, Spec: v1.PodSpec{Containers: []v1.Container{{Name: "test-pod-container", Image: "tomcat:9.0.34"}}}},
		},
		"skip delete event for namespace not configured": {
			// delete operation not allowed for Pod in test namespace so event should be skipped
			GVR:       schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
			Kind:      "Pod",
			Namespace: "test",
			Specs:     &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod-delete"}, Spec: v1.PodSpec{Containers: []v1.Container{{Name: "test-pod-container", Image: "tomcat:9.0.34"}}}},
		},
		"skip delete event for resources not added in test_config": {
			// delete operation not allowed for Pod resources so event should be skipped
			GVR:       schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"},
			Kind:      "Service",
			Namespace: "test",
			Specs:     &v1.Service{ObjectMeta: metav1.ObjectMeta{Name: "test-service-delete"}},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			resource := utils.GVRToString(test.GVR)
			// checking if delete operation is skipped
			isAllowed := utils.CheckOperationAllowed(utils.AllowedEventKindsMap, test.Namespace, resource, config.DeleteEvent)
			assert.Equal(t, isAllowed, false)
		})
	}
	// Resetting original configuration as per test_config
	defer delete(utils.AllowedEventKindsMap, utils.EventKind{Resource: "v1/pods", Namespace: "dummy", EventType: "delete"})
	utils.AllowedEventKindsMap[utils.EventKind{Resource: "v1/pods", Namespace: "all", EventType: "delete"}] = true
	// Resetting original configuration as per test_config adding service resource
	for _, event := range events {
		utils.AllowedEventKindsMap[utils.EventKind{Resource: "v1/services", Namespace: "all", EventType: event}] = true
	}
}

func (c *context) testDeleteEvent(t *testing.T) {
	events := []config.EventType{"update", "error", "create"}

	// Modifying AllowedEventKindsMap to remove events other than delete for pod resource
	for _, event := range events {
		delete(utils.AllowedEventKindsMap, utils.EventKind{Resource: "v1/pods", Namespace: "all", EventType: event})
	}
	// test scenarios
	tests := map[string]testutils.DeleteObjects{
		"perform delete operation and configure BotKube to listen only delete events": {
			GVR:       schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
			Kind:      "Pod",
			Namespace: "test",
			Specs:     &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod-delete"}, Spec: v1.PodSpec{Containers: []v1.Container{{Name: "test-pod-container", Image: "tomcat:9.0.34"}}}},
			ExpectedSlackMessage: testutils.SlackMessage{
				Attachments: []slack.Attachment{{Color: "danger", Title: "v1/pods deleted", Fields: []slack.AttachmentField{{Value: "Pod *test/test-pod-delete* has been deleted in *test-cluster-1* cluster\n", Short: false}}, Footer: "BotKube"}},
			},
			ExpectedWebhookPayload: testutils.WebhookPayload{
				EventMeta:   notify.EventMeta{Kind: "Pod", Name: "test-pod-delete", Namespace: "test", Cluster: "test-cluster-1"},
				EventStatus: notify.EventStatus{Type: "delete", Level: "critical", Reason: "", Error: ""},
				Summary:     "Pod *test/test-pod-delete* has been deleted in *test-cluster-1* cluster\n",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			resource := utils.GVRToString(test.GVR)
			isAllowed := utils.CheckOperationAllowed(utils.AllowedEventKindsMap, test.Namespace, resource, config.DeleteEvent)
			assert.Equal(t, isAllowed, true)

			testutils.DeleteResource(t, test)

			if c.TestEnv.Config.Communications.Slack.Enabled {
				for i := range c.Config.Communications.Slack.AccessBindings {
					// Get last seen slack message
					lastSeenMsg := c.GetLastSeenSlackMessage(i + 1)

					// Convert text message into Slack message structure
					m := slack.Message{}
					err := json.Unmarshal([]byte(*lastSeenMsg), &m)
					if len(m.Attachments) != 0 {
						m.Attachments[0].Ts = ""
					}
					assert.NoError(t, err, "message should decode properly")
					// since same message is sent to all the channels, we are comparing
					// that the each new message received on slack must be configured under AccessBindings,
					// and also  all new messages must be same as expacted one\
					validChannel := utils.Contains(utils.GetAllChannels(&c.TestEnv.Config.Communications.Slack.AccessBindings), m.Channel)
					assert.Equal(t, validChannel, true)
					assert.Equal(t, test.ExpectedSlackMessage.Attachments, m.Attachments)
				}
			}

			if c.TestEnv.Config.Communications.Webhook.Enabled {
				// Get last seen webhook payload
				lastSeenPayload := c.GetLastReceivedPayload()
				assert.Equal(t, test.ExpectedWebhookPayload.EventMeta, lastSeenPayload.EventMeta)
				assert.Equal(t, test.ExpectedWebhookPayload.EventStatus, lastSeenPayload.EventStatus)
				assert.Equal(t, test.ExpectedWebhookPayload.Summary, lastSeenPayload.Summary)
			}
		})
	}
	// Resetting original configuration as per test_config
	for _, event := range events {
		utils.AllowedEventKindsMap[utils.EventKind{Resource: "v1/pods", Namespace: "all", EventType: event}] = true
	}
}

// Run tests
func (c *context) Run(t *testing.T) {
	t.Run("skip delete event", c.testSKipDeleteEvent)
	t.Run("delete resource", c.testDeleteEvent)
}

// E2ETests runs delete notification tests
func E2ETests(testEnv *env.TestEnv) env.E2ETest {
	return &context{
		testEnv,
	}
}
