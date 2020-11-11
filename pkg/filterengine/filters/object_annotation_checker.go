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

package filters

import (
	"github.com/infracloudio/botkube/pkg/events"
	"github.com/infracloudio/botkube/pkg/filterengine"
	"github.com/infracloudio/botkube/pkg/log"
	"github.com/infracloudio/botkube/pkg/utils"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// DisableAnnotation is the object disable annotation
	DisableAnnotation string = "botkube.io/disable"
	// ChannelAnnotation is the multichannel support annotation
	ChannelAnnotation string = "botkube.io/channel"
)

// ObjectAnnotationChecker add recommendations to the event object if pod created without any labels
type ObjectAnnotationChecker struct {
	Description string
}

// Register filter
func init() {
	filterengine.DefaultFilterEngine.Register(ObjectAnnotationChecker{
		Description: "Checks if annotations botkube.io/* present in object specs and filters them.",
	})
}

// Run filters and modifies event struct
func (f ObjectAnnotationChecker) Run(object interface{}, event *events.Event) {

	// get objects metadata
	obj := utils.GetObjectMetaData(object)

	// Check annotations in object
	if isObjectNotifDisabled(obj) {
		event.Skip = true
		log.Debug("Object Notification Disable through annotations")
	}

	if channel, ok := reconfigureChannel(obj); ok {
		var channelList []string
		channelList = append(channelList, channel)
		event.SlackChannels = channelList
		event.MattermostChannels = channelList
		log.Debugf("Redirecting Event Notifications to channel: %s", channel)
	}

	log.Debug("Object annotations filter successful!")
}

// Describe filter
func (f ObjectAnnotationChecker) Describe() string {
	return f.Description
}

// isObjectNotifDisabled checks annotation botkube.io/disable
// annotation botkube.io/disable disables the event notifications from objects
func isObjectNotifDisabled(obj metaV1.ObjectMeta) bool {

	if obj.Annotations[DisableAnnotation] == "true" {
		log.Debug("Skipping Disabled Event Notifications!")
		return true
	}
	return false
}

// reconfigureChannel checks annotation botkube.io/channel
// annotation botkube.io/channel directs event notifications to channels
// based on the channel names present in them
// Note: Add botkube app into the desired channel to receive notifications
func reconfigureChannel(obj metaV1.ObjectMeta) (string, bool) {
	// redirect messages to channels based on annotations
	if channel, ok := obj.Annotations[ChannelAnnotation]; ok {
		return channel, true
	}
	return "", false
}
