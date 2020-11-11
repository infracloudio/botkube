package update

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

func (c *context) testUpdateResource(t *testing.T) {
	// Test cases
	tests := map[string]testutils.UpdateObjects{
		"update resource when IncludeDiff is set to false": {
			// Diff message should not be generated in Attachment if IncludeDiff field is false
			GVR:       schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
			Kind:      "Pod",
			Namespace: "test",
			Specs:     &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod-update-diff-false"}, Spec: v1.PodSpec{Containers: []v1.Container{{Name: "test-pod-container", Image: "tomcat:9.0.34"}}}},
			ExpectedSlackMessage: testutils.SlackMessage{
				Attachments: []slack.Attachment{{Color: "warning", Title: "v1/pods updated", Fields: []slack.AttachmentField{{Value: "Pod *test/test-pod-update-diff-false* has been updated in *test-cluster-1* cluster\n", Short: false}}, Footer: "BotKube"}},
			},
			Patch: []byte(`{
				"apiVersion": "v1",
				"kind": "Pod",
				"metadata": {
				  "name": "test-pod-update-diff-false",
				  "namespace": "test"
				},
				"spec": {
				  "containers": [
					{
					  "name": "test-pod-container",
					  "image": "tomcat:8.0"
					}
				  ]
				}
			  }
			  `),
			UpdateSetting: config.UpdateSetting{Fields: []string{"spec.containers[*].image"}, IncludeDiff: false},
			Diff:          "spec.containers[*].image:\n\t-: tomcat:9.0.34\n\t+: tomcat:8.0\n",
			ExpectedWebhookPayload: testutils.WebhookPayload{
				EventMeta:   notify.EventMeta{Kind: "Pod", Name: "test-pod-update-diff-false", Namespace: "test", Cluster: "test-cluster-1"},
				EventStatus: notify.EventStatus{Type: "update", Level: "warn", Reason: "", Error: "", Messages: []string(nil)},
				Summary:     "Pod *test/test-pod-update-diff-false* has been updated in *test-cluster-1* cluster\n",
			},
		},
		"create and update pod in configured namespace": {
			// Diff message generated in Attachment if IncludeDiff field is true
			GVR:       schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
			Kind:      "Pod",
			Namespace: "test",
			Specs:     &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod-update"}, Spec: v1.PodSpec{Containers: []v1.Container{{Name: "test-pod-container", Image: "tomcat:9.0.34"}}}},
			ExpectedSlackMessage: testutils.SlackMessage{
				Attachments: []slack.Attachment{{Color: "warning", Title: "v1/pods updated", Fields: []slack.AttachmentField{{Value: "Pod *test/test-pod-update* has been updated in *test-cluster-1* cluster\n```\nspec.containers[*].image:\n\t-: tomcat:9.0.34\n\t : tomcat:8.0\n\n```", Short: false}}, Footer: "BotKube"}},
			},
			Patch: []byte(`{
				"apiVersion": "v1",
				"kind": "Pod",
				"metadata": {
				  "name": "test-pod-update",
				  "namespace": "test"
				},
				"spec": {
				  "containers": [
					{
					  "name": "test-pod-container",
					  "image": "tomcat:8.0"
					}
				  ]
				}
			  }
			  `),
			UpdateSetting: config.UpdateSetting{Fields: []string{"spec.containers[*].image"}, IncludeDiff: true},
			Diff:          "spec.containers[*].image:\n\t-: tomcat:9.0.34\n\t+: tomcat:8.0\n",
			ExpectedWebhookPayload: testutils.WebhookPayload{
				EventMeta:   notify.EventMeta{Kind: "Pod", Name: "test-pod-update", Namespace: "test", Cluster: "test-cluster-1"},
				EventStatus: notify.EventStatus{Type: "update", Level: "warn", Reason: "", Error: "", Messages: []string{"spec.containers[*].image:\n\t-: tomcat:9.0.34\n\t+: tomcat:8.0\n"}},
				Summary:     "Pod *test/test-pod-update* has been updated in *test-cluster-1* cluster\n```\nspec.containers[*].image:\n\t-: tomcat:9.0.34\n\t+: tomcat:8.0\n\n```",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			resource := utils.GVRToString(test.GVR)
			// checking if update operation is true
			isAllowed := utils.AllowedEventKindsMap[utils.EventKind{
				Resource:  resource,
				Namespace: "all",
				EventType: config.UpdateEvent}] ||
				utils.AllowedEventKindsMap[utils.EventKind{
					Resource:  resource,
					Namespace: test.Namespace,
					EventType: config.UpdateEvent}]
			assert.Equal(t, isAllowed, true)
			// modifying the update setting value as per testcases
			utils.AllowedUpdateEventsMap[utils.KindNS{Resource: "v1/pods", Namespace: "all"}] = test.UpdateSetting
			// getting the updated and old object
			oldObj, newObj := testutils.UpdateResource(t, test)
			updateMsg := utils.Diff(oldObj.Object, newObj.Object, test.UpdateSetting)
			assert.Equal(t, test.Diff, updateMsg)
			// Inject an event into the fake client.
			if c.TestEnv.Config.Communications.Slack.Enabled {
				for i := range c.Config.Communications.Slack.AccessBindings {
					// Get last seen slack message
					lastSeenMsg := c.GetLastSeenSlackMessage(i + 1)
					// Convert text message into Slack message structure
					m := slack.Message{}
					err := json.Unmarshal([]byte(*lastSeenMsg), &m)
					assert.NoError(t, err, "message should decode properly")
					// since same message is sent to all the channels, we are comparing
					// that the each new message received on slack must be configured under AccessBindings,
					// and also  all new messages must be same as expacted one\
					validChannel := utils.Contains(utils.GetAllChannels(&c.TestEnv.Config.Communications.Slack.AccessBindings), m.Channel)
					assert.Equal(t, validChannel, true)
					if len(m.Attachments) != 0 {
						m.Attachments[0].Ts = ""
					}
					assert.Equal(t, test.ExpectedSlackMessage.Attachments, m.Attachments)
				}
			}

			if c.TestEnv.Config.Communications.Webhook.Enabled {
				// Get last seen webhook payload
				lastSeenPayload := c.GetLastReceivedPayload()
				t.Logf("LastSeenPayload :%#v", lastSeenPayload)
				assert.Equal(t, test.ExpectedWebhookPayload.EventMeta, lastSeenPayload.EventMeta)
				assert.Equal(t, test.ExpectedWebhookPayload.EventStatus, lastSeenPayload.EventStatus)
				assert.Equal(t, test.ExpectedWebhookPayload.Summary, lastSeenPayload.Summary)
			}
		})
	}
}

// Run tests
func (c *context) Run(t *testing.T) {
	t.Run("update resource", c.testUpdateResource)
	t.Run("skip update event", c.testSKipUpdateEvent)
	t.Run("skip update event for wrong setting", c.testSkipWrongSetting)
}

// E2ETests runs create notification tests
func E2ETests(testEnv *env.TestEnv) env.E2ETest {
	return &context{
		testEnv,
	}
}

func (c *context) testSKipUpdateEvent(t *testing.T) {
	// Modifying AllowedEventKindsMap configure dummy namespace for update event and ignore all
	utils.AllowedEventKindsMap[utils.EventKind{Resource: "v1/pods", Namespace: "dummy", EventType: "update"}] = true
	delete(utils.AllowedEventKindsMap, utils.EventKind{Resource: "v1/pods", Namespace: "all", EventType: "update"})
	// reset to original test config
	defer delete(utils.AllowedEventKindsMap, utils.EventKind{Resource: "v1/pods", Namespace: "dummy", EventType: "update"})

	// test scenarios
	tests := map[string]testutils.UpdateObjects{
		"skip update event for namespaces not configured": {
			// update operation not allowed for Pod in test namespace so event should be skipped
			GVR:       schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
			Kind:      "Pod",
			Namespace: "test",
			Specs:     &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod-update"}, Spec: v1.PodSpec{Containers: []v1.Container{{Name: "test-pod-container", Image: "tomcat:9.0.34"}}}},
		},
		"skip update event for resources not added": {
			// update operation not allowed for namespaces in test_config so event should be skipped
			GVR:   schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"},
			Kind:  "Namespace",
			Specs: &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "abc"}},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			resource := utils.GVRToString(test.GVR)
			// checking if update operation is true
			isAllowed := utils.CheckOperationAllowed(utils.AllowedEventKindsMap, test.Namespace, resource, config.UpdateEvent)
			assert.Equal(t, isAllowed, false)
		})
	}
	// Resetting original configuration as per test_config
	utils.AllowedEventKindsMap[utils.EventKind{Resource: "v1/pods", Namespace: "all", EventType: "update"}] = true
}

func (c *context) testSkipWrongSetting(t *testing.T) {
	// test scenarios
	tests := map[string]testutils.UpdateObjects{
		"skip update event for wrong updateSettings value": {
			// update event given with wrong value of updateSettings which doesn't exist would be skipped
			GVR:       schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
			Kind:      "Pod",
			Namespace: "test",
			Specs:     &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod-update-skip"}, Spec: v1.PodSpec{Containers: []v1.Container{{Name: "test-pod-container", Image: "tomcat:9.0.34"}}}},
			Patch: []byte(`{
				"apiVersion": "v1",
				"kind": "Pod",
				"metadata": {
				  "name": "test-pod-update-skip",
				  "namespace": "test"
				},
				"spec": {
				  "containers": [
					{
					  "name": "test-pod-container",
					  "image": "tomcat:8.0"
					}
				  ]
				}
			  }
			  `),
			// adding wrong field
			UpdateSetting: config.UpdateSetting{Fields: []string{"spec.invalid"}, IncludeDiff: true},
			// diff calcuted should be empty because of error
			Diff: "",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			resource := utils.GVRToString(test.GVR)
			// checking if update operation is true
			isAllowed := utils.CheckOperationAllowed(utils.AllowedEventKindsMap, test.Namespace, resource, config.UpdateEvent)
			assert.Equal(t, isAllowed, true)
			// modifying the update setting value as per testcases
			utils.AllowedUpdateEventsMap[utils.KindNS{Resource: "v1/pods", Namespace: "all"}] = test.UpdateSetting
			// getting the updated and old object
			oldObj, newObj := testutils.UpdateResource(t, test)
			updateMsg := utils.Diff(oldObj.Object, newObj.Object, test.UpdateSetting)
			assert.Equal(t, test.Diff, updateMsg)
		})
	}
}
