package filters

import (
	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/events"
	"github.com/infracloudio/botkube/pkg/filterengine"
	"github.com/infracloudio/botkube/pkg/log"
	"github.com/infracloudio/botkube/pkg/utils"
)

type EventReasonChecker struct {
}

// Register filter
func init() {
	filterengine.DefaultFilterEngine.Register(EventReasonChecker{})
}

// Describe filter
func (f EventReasonChecker) Describe() string {
	return "Checks event's reason."
}

// Run filer and modifies event struct
func (f EventReasonChecker) Run(object interface{}, event *events.Event) {

	// Check for Event object
	if utils.GetObjectTypeMetaData(object).Kind != "Event" {
		return
	}

	// load config.yaml
	botkubeConfig, err := config.New()
	if err != nil {
		log.Errorf("Error in loading configuration. %s", err.Error())
		return
	}
	if botkubeConfig == nil {
		log.Errorf("Error in loading configuration.")
		return
	}

	for _, resource := range botkubeConfig.Resources {
		if resource.Name == event.Resource {
			if !isReasonIncluded(event.Reason, resource.Reasons) {
				event.Skip = true
				break
			}

			if isReasonIgnored(event.Reason, resource.Reasons) {
				event.Skip = true
				break
			}
		}
	}
}

func isReasonIncluded(reason string, reasons config.Reasons) bool {
	// empty includes is equal to "all"
	if len(reasons.Include) == 0 {
		return true
	}
	for _, v := range reasons.Include {
		if v == reason || v == "all" {
			return true
		}
	}
	return false
}

func isReasonIgnored(reason string, reasons config.Reasons) bool {
	for _, v := range reasons.Ignore {
		if reason == v {
			return true
		}
	}
	return false
}
