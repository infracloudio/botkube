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
	"net/http"
	"os"

	"github.com/spf13/cobra"

	"github.com/infracloudio/botkube/pkg/audit"
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
	Path               = "/v1/audit"
)

var (
	// rootCmd represents the base command when called without any subcommands
	rootCmd = &cobra.Command{
		Use:   "botkube",
		Short: "Monitor Kubernetes resource lifecycle and audit events.",
	}

	botkubeController = &cobra.Command{
		Use:   "controller",
		Short: "Start watcher on resource lifecycle events and listen for incoming message.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return startController()
		},
	}

	// containersCmd represents the containers command
	auditWebhook = &cobra.Command{
		Use:   "auditwebhook",
		Short: "Start watcher on audit events, filter and send them to configured sink and notifiers.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return startAuditWhServer()
		},
	}
)

func init() {
	rootCmd.AddCommand(botkubeController)
	rootCmd.AddCommand(auditWebhook)
}

func main() {
	// Prometheus metrics
	metricsPort, exists := os.LookupEnv("METRICS_PORT")
	if !exists {
		metricsPort = defaultMetricsPort
	}
	go metrics.ServeMetrics(metricsPort)

	if err := rootCmd.Execute(); err != nil {
		log.Error(err)
		os.Exit(1)
	}
}

func startController() error {
	log.Info("Starting controller")
	conf, err := config.New()
	if err != nil {
		return fmt.Errorf("Error in loading configuration. Error:%s", err.Error())
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

	notifiers := notify.ListNotifiers(conf.Communications)
	log.Infof("Notifier List: config=%#v list=%#v\n", conf.Communications, notifiers)
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
	return nil
}

func startAuditWhServer() error {
	config := config.NewAuditServerConfig()
	log.Infof("Started accepting requests on port=%s", config.Port)
	whHandler, err := audit.NewWebhookHandler()
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc(Path, whHandler.HandlePost)
	return http.ListenAndServeTLS(":"+config.Port, config.TLSCert, config.TLSKey, nil)
}
