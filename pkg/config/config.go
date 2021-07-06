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

package config

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

const (
	// CreateEvent when resource is created
	CreateEvent EventType = "create"
	// UpdateEvent when resource is updated
	UpdateEvent EventType = "update"
	// DeleteEvent when resource deleted
	DeleteEvent EventType = "delete"
	// ErrorEvent on errors in resources
	ErrorEvent EventType = "error"
	// WarningEvent for warning events
	WarningEvent EventType = "warning"
	// NormalEvent for Normal events
	NormalEvent EventType = "normal"
	// InfoEvent for insignificant Info events
	InfoEvent EventType = "info"
	// AllEvent to watch all events
	AllEvent EventType = "all"
	// ShortNotify is the Default NotifType
	ShortNotify NotifType = "short"
	// LongNotify for short events notification
	LongNotify NotifType = "long"

	// Info level
	Info Level = "info"
	// Warn level
	Warn Level = "warn"
	// Debug level
	Debug Level = "debug"
	// Error level
	Error Level = "error"
	// Critical level
	Critical Level = "critical"

	// SlackBot bot platform
	SlackBot BotPlatform = "slack"
	// MattermostBot bot platform
	MattermostBot BotPlatform = "mattermost"
	// TeamsBot bot platform
	TeamsBot BotPlatform = "teams"
	// DiscordBot bot Platform
	DiscordBot BotPlatform = "discord"
)

// EventType to watch
type EventType string

// Level type to store event levels
type Level string

// BotPlatform supported by BotKube
type BotPlatform string

// ResourceConfigFileName is a name of BotKube resource configuration file
var ResourceConfigFileName = "resource_config.yaml"

// CommunicationConfigFileName is a name of BotKube communication configuration file
var CommunicationConfigFileName = "comm_config.yaml"

// Notify flag to toggle event notification
var Notify = true

// NotifType to change notification type
type NotifType string

// Config structure of configuration yaml file
type Config struct {
	Resources       []Resource
	Recommendations bool
	Communications  CommunicationsConfig
	Settings        Settings
}

// Communications contains communication config
type Communications struct {
	Communications CommunicationsConfig
}

// Resource contains resources to watch
type Resource struct {
	Name          string
	Namespaces    Namespaces
	Events        []EventType
	Reasons       Reasons
	UpdateSetting UpdateSetting `yaml:"updateSetting"`
}

//UpdateSetting struct defines updateEvent fields specification
type UpdateSetting struct {
	Fields      []string
	IncludeDiff bool `yaml:"includeDiff"`
}

// Namespaces contains namespaces to include and ignore
// Include contains a list of namespaces to be watched,
//  - "all" to watch all the namespaces
// Ignore contains a list of namespaces to be ignored when all namespaces are included
// It is an optional (omitempty) field which is tandem with Include [all]
// It can also contain a * that would expand to zero or more arbitrary characters
// example : include [all], ignore [x,y,secret-ns-*]
type Namespaces struct {
	Include []string
	Ignore  []string `yaml:",omitempty"`
}

// Reasons contains reasons to include and ignore
// Include contains a list of reasons to be watched,
//  - "all" to watch all the reasons
// Ignore contains a list of reasons to be ignored when all reasons are included
// It is an optional (omitempty) field which is tandem with Include [all]
type Reasons struct {
	Include []string
	Ignore  []string `yaml:",omitempty"`
}

// CommunicationsConfig channels to send events to
type CommunicationsConfig struct {
	Slack         Slack
	Mattermost    Mattermost
	Discord       Discord
	Webhook       Webhook
	Teams         Teams
	ElasticSearch ElasticSearch
}

// Slack configuration to authentication and send notifications
type Slack struct {
	Enabled   bool
	Channel   string
	NotifType NotifType `yaml:",omitempty"`
	Token     string    `yaml:",omitempty"`
}

// ElasticSearch config auth settings
type ElasticSearch struct {
	Enabled       bool
	Username      string
	Password      string `yaml:",omitempty"`
	Server        string
	SkipTLSVerify bool       `yaml:"skipTLSVerify"`
	AWSSigning    AWSSigning `yaml:"awsSigning"`
	Index         Index
}

// AWSSigning contains AWS configurations
type AWSSigning struct {
	Enabled   bool
	AWSRegion string `yaml:"awsRegion"`
	RoleArn   string `yaml:"roleArn"`
}

// Index settings for ELS
type Index struct {
	Name     string
	Type     string
	Shards   int
	Replicas int
}

// Mattermost configuration to authentication and send notifications
type Mattermost struct {
	Enabled   bool
	BotName   string `yaml:"botName"`
	URL       string
	Token     string
	Team      string
	Channel   string
	NotifType NotifType `yaml:",omitempty"`
}

// Teams creds for authentication with MS Teams
type Teams struct {
	Enabled     bool
	AppID       string `yaml:"appID,omitempty"`
	AppPassword string `yaml:"appPassword,omitempty"`
	Team        string
	Port        string
	MessagePath string
	NotifType   NotifType `yaml:",omitempty"`
}

// Discord configuration for authentication and send notifications
type Discord struct {
	Enabled   bool
	Token     string
	BotID     string
	Channel   string
	NotifType NotifType `yaml:",omitempty"`
}

// Webhook configuration to send notifications
type Webhook struct {
	Enabled bool
	URL     string
}

// Kubectl configuration for executing commands inside cluster
type Kubectl struct {
	Enabled          bool
	Commands         Commands
	DefaultNamespace string `yaml:"defaultNamespace"`
	RestrictAccess   bool   `yaml:"restrictAccess"`
}

// Commands allowed in bot
type Commands struct {
	Verbs     []string
	Resources []string
}

// Settings for multicluster support
type Settings struct {
	ClusterName     string
	Kubectl         Kubectl
	ConfigWatcher   bool
	UpgradeNotifier bool `yaml:"upgradeNotifier"`
}

func (eventType EventType) String() string {
	return string(eventType)
}

// NewCommunicationsConfig return new communication config object
func NewCommunicationsConfig() (*Communications, error) {
	c := &Communications{}
	configPath := os.Getenv("CONFIG_PATH")
	communicationConfigFilePath := filepath.Join(configPath, CommunicationConfigFileName)
	communicationConfigFile, err := os.Open(communicationConfigFilePath)
	defer communicationConfigFile.Close()
	if err != nil {
		return c, err
	}

	b, err := ioutil.ReadAll(communicationConfigFile)
	if err != nil {
		return c, err
	}

	if len(b) != 0 {
		yaml.Unmarshal(b, c)
	}
	return c, nil
}

// New returns new Config
func New() (*Config, error) {
	c := &Config{}
	configPath := os.Getenv("CONFIG_PATH")
	resourceConfigFilePath := filepath.Join(configPath, ResourceConfigFileName)
	resourceConfigFile, err := os.Open(resourceConfigFilePath)
	defer resourceConfigFile.Close()
	if err != nil {
		return c, err
	}

	b, err := ioutil.ReadAll(resourceConfigFile)
	if err != nil {
		return c, err
	}

	if len(b) != 0 {
		yaml.Unmarshal(b, c)
	}

	comm, err := NewCommunicationsConfig()
	if err != nil {
		return nil, err
	}
	c.Communications = comm.Communications

	return c, nil
}
