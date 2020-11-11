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

package controller

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"time"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/events"
	"github.com/infracloudio/botkube/pkg/filterengine"

	// Register filters
	_ "github.com/infracloudio/botkube/pkg/filterengine/filters"
	"github.com/infracloudio/botkube/pkg/log"
	"github.com/infracloudio/botkube/pkg/notify"
	"github.com/infracloudio/botkube/pkg/utils"

	"github.com/fsnotify/fsnotify"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
)

const (
	controllerStartMsg = "...and now my watch begins for cluster '%s'! :crossed_swords:"
	controllerStopMsg  = "My watch has ended for cluster '%s'!\nPlease send `@BotKube notifier start` to enable notification once BotKube comes online."
	configUpdateMsg    = "Looks like the configuration is updated for cluster '%s'. I shall halt my watch till I read it."
)

var eventGVR = schema.GroupVersionResource{
	Version:  "v1",
	Resource: "events",
}

var startTime time.Time

// RegisterInformers creates new informer controllers to watch k8s resources
func RegisterInformers(c *config.Config, notifiers []notify.Notifier) {
	sendMessage(c, notifiers, fmt.Sprintf(controllerStartMsg, c.Settings.ClusterName))
	startTime = time.Now()

	// Start config file watcher if enabled
	if c.Settings.ConfigWatcher {
		go configWatcher(c, notifiers)
	}

	// Register informers for resource lifecycle events
	if len(c.Resources) > 0 {
		log.Info("Registering resource lifecycle informer")
		for _, r := range c.Resources {
			if _, ok := utils.ResourceInformerMap[r.Name]; !ok {
				continue
			}
			log.Infof("Adding informer for resource:%s", r.Name)
			utils.ResourceInformerMap[r.Name].AddEventHandler(registerEventHandlers(c, notifiers, r.Name, r.Events))
		}
	}

	// Register informers for k8s events
	log.Infof("Registering kubernetes events informer for types: %+v", config.WarningEvent.String())
	log.Infof("Registering kubernetes events informer for types: %+v", config.NormalEvent.String())
	utils.DynamicKubeInformerFactory.ForResource(eventGVR).Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			var eventObj coreV1.Event
			err := utils.TransformIntoTypedObject(obj.(*unstructured.Unstructured), &eventObj)
			if err != nil {
				log.Errorf("Unable to transform object type: %v, into type: %v", reflect.TypeOf(obj), reflect.TypeOf(eventObj))
			}
			_, err = cache.MetaNamespaceKeyFunc(obj)
			if err != nil {
				log.Errorf("Failed to get MetaNamespaceKey from event resource")
				return
			}

			// Find involved object type
			gvr, err := utils.GetResourceFromKind(eventObj.InvolvedObject.GroupVersionKind())
			if err != nil {
				log.Errorf("Failed to get involved object: %v", err)
				return
			}
			switch strings.ToLower(eventObj.Type) {
			case config.WarningEvent.String():
				// Send WarningEvent as ErrorEvents
				sendEvent(obj, nil, c, notifiers, utils.GVRToString(gvr), config.ErrorEvent)
			case config.NormalEvent.String():
				// Send NormalEvent as Insignificant InfoEvent
				sendEvent(obj, nil, c, notifiers, utils.GVRToString(gvr), config.InfoEvent)
			}
		},
	})
	stopCh := make(chan struct{})
	defer close(stopCh)

	utils.DynamicKubeInformerFactory.Start(stopCh)

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL, syscall.SIGQUIT, syscall.SIGSTOP)

	<-sigterm
	sendMessage(c, notifiers, fmt.Sprintf(controllerStopMsg, c.Settings.ClusterName))
	// Sleep for some time to send termination notification
	time.Sleep(5 * time.Second)
}

func registerEventHandlers(c *config.Config, notifiers []notify.Notifier, resourceType string, events []config.EventType) (handlerFns cache.ResourceEventHandlerFuncs) {
	for _, event := range events {
		if event == config.AllEvent || event == config.CreateEvent {
			handlerFns.AddFunc = func(obj interface{}) {
				log.Debugf("Processing add to %v", resourceType)
				sendEvent(obj, nil, c, notifiers, resourceType, config.CreateEvent)
			}
		}

		if event == config.AllEvent || event == config.UpdateEvent {
			handlerFns.UpdateFunc = func(old, new interface{}) {
				log.Debugf("Processing update to %v\n Object: %+v\n", resourceType, new)
				sendEvent(new, old, c, notifiers, resourceType, config.UpdateEvent)
			}
		}

		if event == config.AllEvent || event == config.DeleteEvent {
			handlerFns.DeleteFunc = func(obj interface{}) {
				log.Debugf("Processing delete to %v", resourceType)
				sendEvent(obj, nil, c, notifiers, resourceType, config.DeleteEvent)
			}
		}
	}
	return handlerFns
}

func sendEvent(obj, oldObj interface{}, c *config.Config, notifiers []notify.Notifier, resource string, eventType config.EventType) {
	// Filter namespaces
	objectMeta := utils.GetObjectMetaData(obj)

	switch eventType {
	case config.InfoEvent:
		// Skip if ErrorEvent is not configured for the resource
		if !utils.CheckOperationAllowed(utils.AllowedEventKindsMap, objectMeta.Namespace, resource, config.ErrorEvent) {
			log.Debugf("Ignoring %s to %s/%v in %s namespaces", eventType, resource, objectMeta.Name, objectMeta.Namespace)
			return
		}
	default:
		if !utils.CheckOperationAllowed(utils.AllowedEventKindsMap, objectMeta.Namespace, resource, eventType) {
			log.Debugf("Ignoring %s to %s/%v in %s namespaces", eventType, resource, objectMeta.Name, objectMeta.Namespace)
			return
		}
	}

	log.Debugf("Processing %s to %s/%v in %s namespaces", eventType, resource, objectMeta.Name, objectMeta.Namespace)

	// Check if Notify disabled
	if !config.Notify {
		log.Debug("Skipping notification")
		return
	}

	// Create new event object
	event := events.New(obj, eventType, resource, c.Settings.ClusterName)
	// Skip older events
	if !event.TimeStamp.IsZero() {
		if event.TimeStamp.Before(startTime) {
			log.Debug("Skipping older events")
			return
		}
	}

	// Check for significant Update Events in objects
	if eventType == config.UpdateEvent {
		updateMsg := ""
		// Check if all namespaces allowed
		updateSetting, exist := utils.AllowedUpdateEventsMap[utils.KindNS{Resource: resource, Namespace: "all"}]
		if !exist {
			// Check if specified namespace is allowed
			updateSetting, exist = utils.AllowedUpdateEventsMap[utils.KindNS{Resource: resource, Namespace: objectMeta.Namespace}]
		}
		if exist {
			// Calculate object diff as per the updateSettings
			var oldUnstruct, newUnstruct *unstructured.Unstructured
			var ok bool
			if oldUnstruct, ok = oldObj.(*unstructured.Unstructured); !ok {
				log.Errorf("Failed to typecast object to Unstructured. Skipping event: %#v", event)
			}
			if newUnstruct, ok = obj.(*unstructured.Unstructured); !ok {
				log.Errorf("Failed to typecast object to Unstructured. Skipping event: %#v", event)
			}
			updateMsg = utils.Diff(oldUnstruct.Object, newUnstruct.Object, updateSetting)
		}

		// Send update notification only if fields in updateSetting are changed
		if len(updateMsg) > 0 {
			if updateSetting.IncludeDiff {
				event.Messages = append(event.Messages, updateMsg)
			}
		} else {
			// skipping least significant update
			log.Debug("skipping least significant Update event")
			event.Skip = true
		}
	}

	// determine channel name where to send the corresponding event based on listed namsepace in selected profiles
	if c.Communications.Slack.Enabled {
		for _, accessBinding := range c.Communications.Slack.AccessBindings {
			for _, namespace := range accessBinding.ProfileValue.Namespaces {
				if event.Namespace == namespace {
					event.SlackChannels = append(event.SlackChannels, accessBinding.ChannelName)
				}
			}
		}
	}
	if c.Communications.Mattermost.Enabled {
		for _, accessBinding := range c.Communications.Mattermost.AccessBindings {
			for _, namespace := range accessBinding.ProfileValue.Namespaces {
				if event.Namespace == namespace {
					event.MattermostChannels = append(event.MattermostChannels, accessBinding.ChannelName)
				}
			}
		}
	}
	if c.Communications.Discord.Enabled {
		for _, accessBinding := range c.Communications.Discord.AccessBindings {
			for _, namespace := range accessBinding.ProfileValue.Namespaces {
				if event.Namespace == namespace {
					event.DiscordChannels = append(event.DiscordChannels, accessBinding.ChannelName)
				}
			}
		}
	}

	// Filter events
	event = filterengine.DefaultFilterEngine.Run(obj, event)
	if event.Skip {
		log.Debugf("Skipping event: %#v", event)
		return
	}

	// Skip unpromoted insignificant InfoEvents
	if event.Type == config.InfoEvent {
		log.Debugf("Skipping Insignificant InfoEvent: %#v", event)
		return
	}

	if len(event.Kind) <= 0 {
		log.Warn("sendEvent received event with Kind nil. Hence skipping.")
		return
	}

	// check if Recommendations are disabled
	if !c.Recommendations {
		event.Recommendations = nil
		log.Debug("Skipping Recommendations in Event Notifications")
	}

	// Send event over notifiers
	for _, n := range notifiers {
		go n.SendEvent(event)
	}
}

func sendMessage(c *config.Config, notifiers []notify.Notifier, msg string) {
	if len(msg) <= 0 {
		log.Warn("sendMessage received string with length 0. Hence skipping.")
		return
	}

	// Send message over notifiers
	for _, n := range notifiers {
		go n.SendMessage(msg)
	}
}

func configWatcher(c *config.Config, notifiers []notify.Notifier) {
	configPath := os.Getenv("CONFIG_PATH")
	configFile := filepath.Join(configPath, config.ResourceConfigFileName)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal("Failed to create file watcher ", err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case _, ok := <-watcher.Events:
				if !ok {
					log.Errorf("Error in getting events for config file:%s. Error: %s", configFile, err.Error())
					return
				}
				log.Infof("Config file %s is updated. Hence restarting the Pod", configFile)
				done <- true

			case err, ok := <-watcher.Errors:
				if !ok {
					log.Errorf("Error in getting events for config file:%s. Error: %s", configFile, err.Error())
					return
				}
			}
		}
	}()
	log.Infof("Registering watcher on configfile %s", configFile)
	err = watcher.Add(configFile)
	if err != nil {
		log.Errorf("Unable to register watch on config file:%s. Error: %s", configFile, err.Error())
		return
	}
	<-done
	sendMessage(c, notifiers, fmt.Sprintf(configUpdateMsg, c.Settings.ClusterName))
	// Wait for Notifier to send message
	time.Sleep(5 * time.Second)
	os.Exit(0)
}
