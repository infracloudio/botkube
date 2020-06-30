// Copyright (c) 2020 InfraCloud Technologies
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
	AuditWebhookTLSCert = "/etc/webhook/certs/tls.crt"
	AuditWebhookTLSKey  = "/etc/webhook/certs/tls.key"
	AuditConfigFileName = "audit_config.yaml"
)

type AuditConfig struct {
	ExternalSink ExternalAuditSink `yaml:"externalSink"`
	Notifier     AuditNotifier
}

type ExternalAuditSink struct {
	ElasticSearch ElasticSearch
}

type AuditNotifier struct {
	Enabled     bool
	ClusterName string
	Rules       []AuditRule
}

type AuditRule struct {
	Name        string
	Description string
	Condition   string
	Priority    Level
}

type AuditServerConfig struct {
	TLSKey  string
	TLSCert string
	Port    string
}

func NewAuditConfig() (*AuditConfig, error) {
	c := &AuditConfig{}
	configPath := os.Getenv("CONFIG_PATH")
	auditConfigFilePath := filepath.Join(configPath, AuditConfigFileName)
	resourceConfigFile, err := os.Open(auditConfigFilePath)
	if err != nil {
		return c, err
	}
	defer resourceConfigFile.Close()
	b, err := ioutil.ReadAll(resourceConfigFile)
	if err != nil {
		return c, err
	}

	if len(b) != 0 {
		yaml.Unmarshal(b, c)
	}
	return c, nil
}

func NewAuditServerConfig() *AuditServerConfig {
	// Default values
	conf := &AuditServerConfig{
		TLSKey:  AuditWebhookTLSCert,
		TLSCert: AuditWebhookTLSKey,
		Port:    "8080",
	}
	if v, ok := os.LookupEnv("AUDIT_WEBHOOK_TLS_CERT"); ok {
		conf.TLSCert = v
	}
	if v, ok := os.LookupEnv("AUDIT_WEBHOOK_TLS_KEY"); ok {
		conf.TLSKey = v
	}
	if v, ok := os.LookupEnv("AUDIT_WEBHOOK_PORT"); ok {
		conf.Port = v
	}
	return conf
}
