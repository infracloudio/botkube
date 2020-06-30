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
	"strings"
	"time"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/utils"
	appsV1 "k8s.io/api/apps/v1"
	batchV1 "k8s.io/api/batch/v1"
	coreV1 "k8s.io/api/core/v1"
	networkV1beta1 "k8s.io/api/networking/v1beta1"
	rbacV1 "k8s.io/api/rbac/v1"
)

// Event to store required information from k8s objects
type Event struct {
	Code        string
	Title       string
	Description string
	Kind        string
	Name        string
	Namespace   string
	Messages    []string
	Type        config.EventType
	Reason      string
	Error       string
	Level       config.Level
	Cluster     string
	Channel     string
	TimeStamp   time.Time
	Count       int32
	Action      string
	Skip        bool `json:",omitempty"`

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
func New(object interface{}, eventType config.EventType, kind, clusterName string) Event {
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
	// kind of the object associated with it
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

	if kind != "events" {
		switch eventType {
		case config.ErrorEvent, config.InfoEvent:
			event.Title = strings.Title(fmt.Sprintf("%s %s", kind, eventType.String()))
		default:
			// Events like create, update, delete comes with an extra 'd' at the end
			event.Title = strings.Title(fmt.Sprintf("%s %sd", kind, eventType.String()))
		}
	}

	switch obj := object.(type) {
	case *coreV1.Event:
		event.Reason = obj.Reason
		event.Messages = append(event.Messages, obj.Message)
		event.Kind = obj.InvolvedObject.Kind
		event.Name = obj.InvolvedObject.Name
		event.Namespace = obj.InvolvedObject.Namespace
		event.Level = LevelMap[config.EventType(strings.ToLower(obj.Type))]
		event.Count = obj.Count
		event.Action = obj.Action
		event.TimeStamp = obj.LastTimestamp.Time
	case *coreV1.Pod:
		event.Kind = "Pod"
	case *coreV1.Node:
		event.Kind = "Node"
	case *coreV1.Namespace:
		event.Kind = "Namespace"
	case *coreV1.PersistentVolume:
		event.Kind = "PersistentVolume"
	case *coreV1.PersistentVolumeClaim:
		event.Kind = "PersistentVolumeClaim"
	case *coreV1.ReplicationController:
		event.Kind = "ReplicationController"
	case *coreV1.Service:
		event.Kind = "Service"
	case *coreV1.Secret:
		event.Kind = "Secret"
	case *coreV1.ConfigMap:
		event.Kind = "ConfigMap"

	case *networkV1beta1.Ingress:
		event.Kind = "Ingress"

	case *appsV1.DaemonSet:
		event.Kind = "DaemonSet"
	case *appsV1.ReplicaSet:
		event.Kind = "ReplicaSet"
	case *appsV1.Deployment:
		event.Kind = "Deployment"
	case *appsV1.StatefulSet:
		event.Kind = "StatefulSet"

	case *batchV1.Job:
		event.Kind = "Job"

	case *rbacV1.Role:
		event.Kind = "Role"
	case *rbacV1.RoleBinding:
		event.Kind = "RoleBinding"
	case *rbacV1.ClusterRole:
		event.Kind = "ClusterRole"
	case *rbacV1.ClusterRoleBinding:
		event.Kind = "ClusterRoleBinding"
	}

	return event
}
