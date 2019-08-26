package filters

import (
	"strings"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/events"
	"github.com/infracloudio/botkube/pkg/filterengine"
	log "github.com/infracloudio/botkube/pkg/logging"

	apiV1 "k8s.io/api/core/v1"
)

// ImageTagChecker add recommendations to the event object if latest image tag is used in pod containers
type ImageTagChecker struct {
	Description string
}

// Register filter
func init() {
	filterengine.DefaultFilterEngine.Register(ImageTagChecker{
		Description: "Checks and adds recommendation if 'latest' image tag is used for container image.",
	})
}

// Run filers and modifies event struct
func (f ImageTagChecker) Run(object interface{}, event *events.Event) {
	if event.Kind != "Pod" || event.Type != config.CreateEvent {
		return
	}
	podObj, ok := object.(*apiV1.Pod)
	if !ok {
		return
	}

	// Check image tag in initContainers
	for _, ic := range podObj.Spec.InitContainers {
		images := strings.Split(ic.Image, ":")
		if len(images) == 1 || images[1] == "latest" {
			event.Recommendations = append(event.Recommendations, ":latest tag used in image '"+ic.Image+"' of initContainer '"+ic.Name+"' should be avoided.\n")
		}
	}

	// Check image tag in Containers
	for _, c := range podObj.Spec.Containers {
		images := strings.Split(c.Image, ":")
		if len(images) == 1 || images[1] == "latest" {
			event.Recommendations = append(event.Recommendations, ":latest tag used in image '"+c.Image+"' of Container '"+c.Name+"' should be avoided.\n")
		}
	}
	log.Logger.Debug("Image tag filter successful!")
}

// Describe filter
func (f ImageTagChecker) Describe() string {
	return f.Description
}
