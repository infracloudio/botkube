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

package main

import (
	"fmt"
	"os"

	"github.com/infracloudio/botkube/pkg/bot"
	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/controller"
	"github.com/infracloudio/botkube/pkg/log"
	"github.com/infracloudio/botkube/pkg/metrics"
	"github.com/infracloudio/botkube/pkg/notify"
	"github.com/infracloudio/botkube/pkg/utils"
)

const (
	defaultMetricsPort = "2112"
)

func main() {
	log.Info("Starting controller")
	conf, err := config.New()
	if err != nil {
		log.Fatal(fmt.Sprintf("Error in loading configuration. Error:%s", err.Error()))
	}

	if conf.Communications.Slack.Enabled {
		log.Info("Starting slack bot")
		sb := bot.NewSlackBot(conf)
		go sb.Start()
	}

	if conf.Communications.Mattermost.Enabled {
		log.Info("Starting mattermost bot")
		mb := bot.NewMattermostBot(conf)
		go mb.Start()
	}

	if conf.Communications.Telegram.Enabled {
		log.Info("Starting telegram bot")
		sb := bot.NewTelegramBot(conf)
		go sb.Start()
	}

	// Prometheus metrics
	metricsPort, exists := os.LookupEnv("METRICS_PORT")
	if !exists {
		metricsPort = defaultMetricsPort
	}
	go metrics.ServeMetrics(metricsPort)

	notifiers := listNotifiers(conf)
	// Start upgrade notifier
	if conf.Settings.UpgradeNotifier {
		log.Info("Starting upgrade notifier")
		go controller.UpgradeNotifier(conf, notifiers)
	}

	// Init KubeClient, InformerMap and start controller
	utils.InitKubeClient()
	utils.InitInformerMap(conf)
	utils.InitResourceMap(conf)
	controller.RegisterInformers(conf, notifiers)
}

func listNotifiers(conf *config.Config) []notify.Notifier {
	var notifiers []notify.Notifier
	if conf.Communications.Slack.Enabled {
		notifiers = append(notifiers, notify.NewSlack(conf))
	}
	if conf.Communications.Mattermost.Enabled {
		if notifier, err := notify.NewMattermost(conf); err == nil {
			notifiers = append(notifiers, notifier)
		} else {
			log.Error(fmt.Sprintf("Failed to create Mattermost client. Error: %v", err))
		}
	}
	if conf.Communications.ElasticSearch.Enabled {
		if els, err := notify.NewElasticSearch(conf); err == nil {
			notifiers = append(notifiers, els)
		} else {
			log.Error(fmt.Sprintf("Failed to create els client. Error: %v", err))
		}
	}
	if conf.Communications.Webhook.Enabled {
		notifiers = append(notifiers, notify.NewWebhook(conf))
	}
	return notifiers
}
