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

package notify

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	v4 "github.com/aws/aws-sdk-go/aws/signer/v4"
	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/events"
	"github.com/infracloudio/botkube/pkg/log"
	"github.com/olivere/elastic"
	"github.com/sha1sum/aws_signing_client"
)

const (
	// indexSuffixFormat is the date format that would be appended to the index name
	indexSuffixFormat = "2006-01-02" // YYYY-MM-DD
	// awsService for the AWS client to authenticate against
	awsService = "es"
)

// ElasticSearch contains auth cred and index setting
type ElasticSearch struct {
	ELSClient *elastic.Client
	Server    string
	Index     string
	Shards    int
	Replicas  int
	Type      string
}

// ElasticSearchEvent contains event information specific to ElasticSearch
type ElasticSearchEvent struct {
	Code      string
	Title     string
	Kind      string
	Name      string
	Namespace string
	Messages  []string
	Type      config.EventType
	Reason    string
	Error     string
	Level     config.Level
	Cluster   string
	TimeStamp time.Time
	Count     int32
	Action    string
	Skip      bool `json:",omitempty"`

	Recommendations []string
	Warnings        []string
}

// NewElasticSearch returns new ElasticSearch object
func NewElasticSearch(c config.ElasticSearch) (Notifier, error) {
	var elsClient *elastic.Client
	var err error
	var creds *credentials.Credentials
	if c.AWSSigning.Enabled {
		// Get credentials from environment variables and create the AWS Signature Version 4 signer
		sess := session.Must(session.NewSession())
		if c.AWSSigning.RoleArn != "" {
			creds = stscreds.NewCredentials(sess, c.AWSSigning.RoleArn)
		} else {
			creds = ec2rolecreds.NewCredentials(sess)
		}

		signer := v4.NewSigner(creds)
		awsClient, err := aws_signing_client.New(signer, nil, awsService, c.AWSSigning.AWSRegion)
		if err != nil {
			return nil, err
		}
		elsClient, err = elastic.NewClient(
			elastic.SetURL(c.Server),
			elastic.SetScheme("https"),
			elastic.SetHttpClient(awsClient),
			elastic.SetSniff(false),
			elastic.SetHealthcheck(false),
			elastic.SetGzip(false),
		)
		if err != nil {
			return nil, err
		}
	} else {
		// create elasticsearch client
		elsClient, err = elastic.NewClient(
			elastic.SetURL(c.Server),
			elastic.SetBasicAuth(c.Username, c.Password),
			elastic.SetSniff(false),
			elastic.SetHealthcheck(false),
			elastic.SetGzip(true),
		)
		if err != nil {
			return nil, err
		}
	}
	return &ElasticSearch{
		ELSClient: elsClient,
		Index:     c.Index.Name,
		Type:      c.Index.Type,
		Shards:    c.Index.Shards,
		Replicas:  c.Index.Replicas,
	}, nil
}

type mapping struct {
	Settings settings `json:"settings"`
}

type settings struct {
	Index index `json:"index"`
}
type index struct {
	Shards   int `json:"number_of_shards"`
	Replicas int `json:"number_of_replicas"`
}

func (e *ElasticSearch) flushIndex(ctx context.Context, event interface{}) error {
	// Construct the ELS Index Name with timestamp suffix
	indexName := e.Index + "-" + time.Now().Format(indexSuffixFormat)
	// Create index if not exists
	exists, err := e.ELSClient.IndexExists(indexName).Do(ctx)
	if err != nil {
		log.Error(fmt.Sprintf("Failed to get index. Error:%s", err.Error()))
		return err
	}
	if !exists {
		// Create a new index.
		mapping := mapping{
			Settings: settings{
				index{
					Shards:   e.Shards,
					Replicas: e.Replicas,
				},
			},
		}
		_, err := e.ELSClient.CreateIndex(indexName).BodyJson(mapping).Do(ctx)
		if err != nil {
			log.Error(fmt.Sprintf("Failed to create index. Error:%s", err.Error()))
			return err
		}
	}

	// Send event to els
	_, err = e.ELSClient.Index().Index(indexName).Type(e.Type).BodyJson(event).Do(ctx)
	if err != nil {
		log.Error(fmt.Sprintf("Failed to post data to els. Error:%s", err.Error()))
		return err
	}
	_, err = e.ELSClient.Flush().Index(indexName).Do(ctx)
	if err != nil {
		log.Error(fmt.Sprintf("Failed to flush data to els. Error:%s", err.Error()))
		return err
	}
	log.Debugf("Event successfully sent to ElasticSearch index %s", indexName)
	return nil
}

// SendEvent sends event notification to slack
func (e *ElasticSearch) SendEvent(event events.Event) (err error) {
	log.Debug(fmt.Sprintf(">> Sending to ElasticSearch: %+v", event))
	ctx := context.Background()
	elasticSearchEvent := &ElasticSearchEvent{
		Code:            event.Code,
		Title:           event.Title,
		Kind:            event.Kind,
		Name:            event.Name,
		Namespace:       event.Namespace,
		Messages:        event.Messages,
		Type:            event.Type,
		Reason:          event.Reason,
		Error:           event.Error,
		Level:           event.Level,
		Cluster:         event.Cluster,
		TimeStamp:       event.TimeStamp,
		Count:           event.Count,
		Action:          event.Action,
		Skip:            event.Skip,
		Recommendations: event.Recommendations,
		Warnings:        event.Warnings,
	}
	// Create index if not exists
	if err := e.flushIndex(ctx, elasticSearchEvent); err != nil {
		return err
	}
	return nil
}

// SendMessage sends message to slack channel
func (e *ElasticSearch) SendMessage(msg string) error {
	return nil
}
