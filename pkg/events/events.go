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

package events

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/infracloudio/botkube/pkg/config"
	log "github.com/infracloudio/botkube/pkg/log"
	"github.com/infracloudio/botkube/pkg/utils"

	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Event to store required information from k8s objects
type Event struct {
	Code               string
	Title              string
	Kind               string
	Name               string
	Namespace          string
	Messages           []string
	Type               config.EventType
	Reason             string
	Error              string
	Level              config.Level
	Cluster            string
	SlackChannels      []string
	DiscordChannels    []string
	MattermostChannels []string
	TimeStamp          time.Time
	Count              int32
	Action             string
	Skip               bool `json:",omitempty"`

	Recommendations []string
	Warnings        []string
}

// LevelMap is a map of event type to Level
var LevelMap map[config.EventType]config.Level

func init() {
	LevelMap = make(map[config.EventType]config.Level)
	LevelMap[config.CreateEvent] = config.Info
	LevelMap[config.UpdateEvent] = config.Warn
	LevelMap[config.DeleteEvent] = config.Critical
	LevelMap[config.ErrorEvent] = config.Error
	LevelMap[config.WarningEvent] = config.Error
}

// New extract required details from k8s object and returns new Event object
func New(object interface{}, eventType config.EventType, resource, clusterName string) Event {
	objectTypeMeta := utils.GetObjectTypeMetaData(object)
	objectMeta := utils.GetObjectMetaData(object)

	event := Event{
		Name:      objectMeta.Name,
		Namespace: objectMeta.Namespace,
		Kind:      objectTypeMeta.Kind,
		Level:     LevelMap[eventType],
		Type:      eventType,
		Cluster:   clusterName,
	}

	// initialize event.TimeStamp with the time of event creation
	// event.TimeStamp is overwritten later based on the type of the event or
	// resource of the object associated with it
	event.TimeStamp = time.Now()

	// Add TimeStamps
	if eventType == config.CreateEvent {
		event.TimeStamp = objectMeta.CreationTimestamp.Time
	}

	if eventType == config.DeleteEvent {
		if objectMeta.DeletionTimestamp != nil {
			event.TimeStamp = objectMeta.DeletionTimestamp.Time
		}
	}

	switch eventType {
	case config.ErrorEvent, config.InfoEvent:
		event.Title = fmt.Sprintf("%s %s", resource, eventType.String())
	default:
		// Events like create, update, delete comes with an extra 'd' at the end
		event.Title = fmt.Sprintf("%s %sd", resource, eventType.String())
	}

	if objectTypeMeta.Kind == "Event" {
		var eventObj coreV1.Event
		err := utils.TransformIntoTypedObject(object.(*unstructured.Unstructured), &eventObj)
		if err != nil {
			log.Errorf("Unable to transform object type: %v, into type: %v", reflect.TypeOf(object), reflect.TypeOf(eventObj))
		}
		event.Reason = eventObj.Reason
		event.Messages = append(event.Messages, eventObj.Message)
		event.Kind = eventObj.InvolvedObject.Kind
		event.Name = eventObj.InvolvedObject.Name
		event.Namespace = eventObj.InvolvedObject.Namespace
		event.Level = LevelMap[config.EventType(strings.ToLower(eventObj.Type))]
		event.Count = eventObj.Count
		event.Action = eventObj.Action
		event.TimeStamp = eventObj.LastTimestamp.Time

	}
	return event
}
