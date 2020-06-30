package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/open-policy-agent/opa/rego"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kaudit "k8s.io/apiserver/pkg/apis/audit"

	"github.com/infracloudio/botkube/pkg/config"
	bkevents "github.com/infracloudio/botkube/pkg/events"
	log "github.com/infracloudio/botkube/pkg/log"
	"github.com/infracloudio/botkube/pkg/notify"
)

type allowedEvent struct {
	event       []byte
	matchedRule config.AuditRule
}

type WebhookHandler struct {
	BotKubeNotifiers []notify.Notifier
	ExternalSink     []ExternalSink
	Config           *config.AuditConfig
}

type EventList struct {
	metav1.TypeMeta
	// +optional
	metav1.ListMeta

	// Preserve JSON for rego eval
	Items []map[string]interface{}
}

func NewWebhookHandler() (*WebhookHandler, error) {
	conf, err := config.NewAuditConfig()
	if err != nil {
		return nil, err
	}
	commConf, err := config.NewCommunicationsConfig()
	if err != nil {
		return nil, err
	}
	extSink := []ExternalSink{}

	if conf.ExternalSink.ElasticSearch.Enabled {
		elsSink, err := notify.NewElasticSearch(conf.ExternalSink.ElasticSearch)
		if err != nil {
			return nil, err
		}
		extSink = append(extSink, elsSink)
	}
	log.Infof("Notifier List: config=%#v list=%#v\n", *commConf, notify.ListNotifiers(commConf.Communications))
	log.Infof("External Sink List: config=%#v list=%#v\n", *conf, extSink)
	return &WebhookHandler{
		BotKubeNotifiers: notify.ListNotifiers(commConf.Communications),
		ExternalSink:     extSink,
		Config:           conf,
	}, nil
}

func (wh *WebhookHandler) sendToExtSink(event []byte) {
	kevent := kaudit.Event{}
	if err := json.Unmarshal(event, &kevent); err != nil {
		log.Errorf("Failed to unmarshal audit event. %v", err.Error())
	}
	for _, s := range wh.ExternalSink {
		if err := s.SendAuditEvent(kevent); err != nil {
			log.Error(err.Error())
		}
	}
}

func (wh *WebhookHandler) HandlePost(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Invalid request method.", 405)
		return
	}
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error(err)
		http.Error(w, fmt.Sprintf("Bad Request. Error: %s", err.Error()), http.StatusBadRequest)
		return
	}
	eventList := EventList{}
	if err := json.Unmarshal(data, &eventList); err != nil {
		log.Error(err)
		http.Error(w, fmt.Sprintf(err.Error()), http.StatusInternalServerError)
		return
	}

	// TODO: maintain workpool and queue for load balancing
	go wh.ParseAndSend(eventList)
}

func (wh *WebhookHandler) ParseAndSend(eventList EventList) {
	events, err := wh.RulesAllowed(eventList, wh.Config.Notifier.Rules)
	if err != nil {
		log.Errorf("Failed to evaluate rules. %+v", err)
		return
	}
	for _, event := range events {
		log.Infof("Sending event %s %#v\n", string(event.event), event.matchedRule)
		bkEvent, err := wh.mapToBotKubeEvent(event)
		if err != nil {
			log.Errorf("Failed to map Audit event to BotKube event. %+v", err)
			continue
		}

		// Send event over notifiers
		for _, n := range wh.BotKubeNotifiers {
			log.Infof("Sending to %#v\n", n)
			go n.SendEvent(bkEvent)
		}
	}
}

func (wh *WebhookHandler) mapToBotKubeEvent(event allowedEvent) (bkevents.Event, error) {
	auditEvent := &kaudit.Event{}
	err := json.Unmarshal(event.event, auditEvent)
	if err != nil {
		return bkevents.Event{}, err
	}

	bkEvent := bkevents.Event{
		Title:       event.matchedRule.Name,
		Description: event.matchedRule.Description,
		Messages:    []string{string(event.event)},
		Type:        config.AuditEvent,
		Level:       event.matchedRule.Priority,
		Cluster:     wh.Config.Notifier.ClusterName,
	}
	if auditEvent.ObjectRef != nil {
		bkEvent.Kind = auditEvent.ObjectRef.Resource
		bkEvent.Name = auditEvent.ObjectRef.Name
		bkEvent.Namespace = auditEvent.ObjectRef.Namespace
	}
	return bkEvent, nil
}

func (wh *WebhookHandler) RulesAllowed(eventList EventList, rules []config.AuditRule) ([]allowedEvent, error) {
	allowedEvents := []allowedEvent{}
	for _, event := range eventList.Items {
		eventJSON, err := json.Marshal(event)
		if err != nil {
			return nil, err
		}
		// TODO: Create sink list
		// Send to external sink (currently only targetting ELS)
		go wh.sendToExtSink(eventJSON)

		d := json.NewDecoder(bytes.NewBuffer(eventJSON))
		d.UseNumber()
		var input interface{}

		if err := d.Decode(&input); err != nil {
			return nil, err
		}

		for _, rule := range rules {
			rego := rego.New(
				rego.Query(rule.Condition),
				rego.Input(input))

			// Evaluate rego expression
			rs, err := rego.Eval(context.TODO())
			if err != nil {
				return nil, err
			}
			if len(rs) == 0 {
				continue
			}
			for _, r := range rs {
				for _, e := range r.Expressions {
					b, ok := e.Value.(bool)
					if !ok {
						continue
					}
					if b != true {
						continue
					}
				}
				allowedEvents = append(allowedEvents, allowedEvent{event: eventJSON, matchedRule: rule})
			}
		}
	}
	return allowedEvents, nil
}
