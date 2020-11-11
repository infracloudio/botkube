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

package bot

import (
	"strings"

	"github.com/bwmarrin/discordgo"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/execute"
	"github.com/infracloudio/botkube/pkg/log"
)

// DiscordBot listens for user's message, execute commands and sends back the response
type DiscordBot struct {
	Token            string
	AllowKubectl     bool
	RestrictAccess   bool
	ClusterName      string
	AccessBindings   []config.AccessBinding
	BotID            string
	DefaultNamespace string
}

// discordMessage contains message details to execute command and send back the result
type discordMessage struct {
	Event         *discordgo.MessageCreate
	BotID         string
	Request       string
	Response      string
	IsAuthChannel bool
	Session       *discordgo.Session
}

// NewDiscordBot returns new Bot object
func NewDiscordBot(c *config.Config) Bot {
	return &DiscordBot{
		Token:            c.Communications.Discord.Token,
		BotID:            c.Communications.Discord.BotID,
		AllowKubectl:     c.Settings.Kubectl.Enabled,
		RestrictAccess:   c.Settings.Kubectl.RestrictAccess,
		ClusterName:      c.Settings.ClusterName,
		AccessBindings:   c.Communications.Discord.AccessBindings,
		DefaultNamespace: c.Settings.Kubectl.DefaultNamespace,
	}
}

// Start starts the DiscordBot websocket connection and listens for messages
func (b *DiscordBot) Start() {
	api, err := discordgo.New("Bot " + b.Token)
	if err != nil {
		log.Error("error creating Discord session,", err)
		return
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	api.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		dm := discordMessage{
			Event:   m,
			BotID:   b.BotID,
			Session: s,
		}

		dm.HandleMessage(b)
	})

	// Open a websocket connection to Discord and begin listening.
	go func() {
		err := api.Open()
		if err != nil {
			log.Error("error opening connection,", err)
			return
		}
	}()

	log.Info("BotKube connected to Discord!")
}

// HandleMessage handles the incoming messages
func (dm *discordMessage) HandleMessage(b *DiscordBot) {
	accessBinding := config.AccessBinding{}
	// Serve only if starts with mention
	if !strings.HasPrefix(dm.Event.Content, "<@!"+dm.BotID+"> ") && !strings.HasPrefix(dm.Event.Content, "<@"+dm.BotID+"> ") {
		return
	}

	// Trim the @BotKube prefix
	if strings.HasPrefix(dm.Event.Content, "<@!"+dm.BotID+"> ") {
		dm.Request = strings.TrimPrefix(dm.Event.Content, "<@!"+dm.BotID+"> ")
	} else if strings.HasPrefix(dm.Event.Content, "<@"+dm.BotID+"> ") {
		dm.Request = strings.TrimPrefix(dm.Event.Content, "<@"+dm.BotID+"> ")
	}

	if len(dm.Request) == 0 {
		return
	}

	for _, accessBind := range b.AccessBindings {
		if accessBind.ChannelName == dm.Event.ChannelID {
			dm.IsAuthChannel = true
			accessBinding = accessBind
			break
		}
	}

	e := execute.NewDefaultExecutor(dm.Request, b.AllowKubectl, b.RestrictAccess, b.DefaultNamespace,
		b.ClusterName, accessBinding.ProfileValue, config.DiscordBot, accessBinding.ChannelName, dm.IsAuthChannel)

	dm.Response = e.Execute()
	dm.Send()
}

func (dm discordMessage) Send() {
	log.Debugf("Discord incoming Request: %s", dm.Request)
	log.Debugf("Discord Response: %s", dm.Response)

	// Upload message as a file if too long
	if len(dm.Response) >= 2000 {
		params := &discordgo.MessageSend{
			Content: dm.Request,
			Files: []*discordgo.File{
				{
					Name:   "Response",
					Reader: strings.NewReader(dm.Response),
				},
			},
		}

		if _, err := dm.Session.ChannelMessageSendComplex(dm.Event.ChannelID, params); err != nil {
			log.Error("Error in uploading file:", err)
		}
		return
	} else if len(dm.Response) == 0 {
		log.Info("Invalid request. Dumping the response")
		return
	}

	if _, err := dm.Session.ChannelMessageSend(dm.Event.ChannelID, dm.Response); err != nil {
		log.Error("Error in sending message:", err)
	}
}
