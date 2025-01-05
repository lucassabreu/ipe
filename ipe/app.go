// Copyright 2014 Claudemiro Alves Feitosa Neto. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ipe

import (
	"errors"
	"expvar"
	"fmt"
	"sync"

	log "github.com/golang/glog"
	"github.com/millerp/ipe/ipe/bus"
)

// An App
type app struct {
	sync.Mutex

	Name                string
	AppID               string
	Key                 string
	Secret              string
	OnlySSL             bool
	ApplicationDisabled bool
	UserEvents          bool
	WebHooks            bool
	URLWebHook          string

	Channels    map[string]*channel
	Connections map[string]*connection

	Stats *expvar.Map

	Bus bus.Bus
}

func newApp(
	name, appID, key, secret string,
	onlySSL, disabled, userEvents,
	webHooks bool, webHookURL string,
	bus bus.Bus,
) *app {
	a := &app{
		Name:                name,
		AppID:               appID,
		Key:                 key,
		Secret:              secret,
		OnlySSL:             onlySSL,
		ApplicationDisabled: disabled,
		UserEvents:          userEvents,
		WebHooks:            webHooks,
		URLWebHook:          webHookURL,
		Bus:                 bus,
	}

	a.Connections = make(map[string]*connection)
	a.Channels = make(map[string]*channel)
	a.Stats = expvar.NewMap(fmt.Sprintf("%s (%s)", a.Name, a.AppID))

	return a
}

// Only Presence channels
func (a *app) PresenceChannels() []*channel {
	var channels []*channel

	for _, c := range a.Channels {
		if c.IsPresence() {
			channels = append(channels, c)
		}
	}

	return channels
}

// Only Private channels
func (a *app) PrivateChannels() []*channel {
	var channels []*channel

	for _, c := range a.Channels {
		if c.IsPrivate() {
			channels = append(channels, c)
		}
	}

	return channels
}

// Only Public channels
func (a *app) PublicChannels() []*channel {
	var channels []*channel

	for _, c := range a.Channels {
		if c.IsPublic() {
			channels = append(channels, c)
		}
	}

	return channels
}

// Disconnect Socket
func (a *app) Disconnect(socketID string) {
	log.Infof("Disconnecting socket %+v", socketID)

	conn, err := a.FindConnection(socketID)

	if err != nil {
		log.Infof("Socket not found, %+v", err)
		return
	}

	// Unsubscribe from channels
	for _, c := range a.Channels {
		if c.IsSubscribed(conn) {
			c.Unsubscribe(a, conn)
		}
	}

	// Remove from app
	a.Lock()
	defer a.Unlock()

	_, exists := a.Connections[conn.SocketID]

	if !exists {
		return
	}

	delete(a.Connections, conn.SocketID)

	a.Stats.Add("TotalConnections", -1)
}

// Connect a new Subscriber
func (a *app) Connect(conn *connection) {
	log.Infof("Adding a new Connection %s to app %s", conn.SocketID, a.Name)
	a.Lock()
	defer a.Unlock()

	a.Connections[conn.SocketID] = conn

	a.Stats.Add("TotalConnections", 1)
}

// Find a Connection on this app
func (a *app) FindConnection(socketID string) (*connection, error) {
	conn, exists := a.Connections[socketID]

	if exists {
		return conn, nil
	}

	return nil, errors.New("Connection not found")
}

// DeleteChannel removes the channel from app
func (a *app) RemoveChannel(c *channel) {
	log.Infof("Remove the channel %s from app %s", c.ChannelID, a.Name)
	a.Lock()
	defer a.Unlock()

	delete(a.Channels, c.ChannelID)

	if c.IsPresence() {
		a.Stats.Add("TotalPresenceChannels", -1)
	}

	if c.IsPrivate() {
		a.Stats.Add("TotalPrivateChannels", -1)
	}

	if c.IsPublic() {
		a.Stats.Add("TotalPublicChannels", -1)
	}

	a.Stats.Add("TotalChannels", -1)
}

// Add a new Channel to this APP
func (a *app) AddChannel(c *channel) {
	log.Infof("Adding a new channel %s to app %s", c.ChannelID, a.Name)

	a.Lock()
	defer a.Unlock()

	a.Channels[c.ChannelID] = c

	if c.IsPresence() {
		a.Stats.Add("TotalPresenceChannels", 1)
	}

	if c.IsPrivate() {
		a.Stats.Add("TotalPrivateChannels", 1)
	}

	if c.IsPublic() {
		a.Stats.Add("TotalPublicChannels", 1)
	}

	a.Stats.Add("TotalChannels", 1)
}

// Returns a Channel from this app
// If not found then the channel is created and added to this app
func (a *app) FindOrCreateChannelByChannelID(n string) *channel {
	c, err := a.FindChannelByChannelID(n)

	if err != nil {
		c = newChannel(n)
		a.AddChannel(c)
	}

	return c
}

// Find the channel by channel ID
func (a *app) FindChannelByChannelID(n string) (*channel, error) {
	c, exists := a.Channels[n]

	if exists {
		return c, nil
	} else {
		c = newChannel(n)
		a.Channels[n] = c
		log.Infof("Channel Created: %s", n)
	}

	return c, nil
}

func (a *app) Publish(c *channel, event rawEvent, ignore string) error {
	a.Stats.Add("TotalUniqueMessages", 1)

	return a.Bus.Publish(bus.Message{
		AppID:    a.AppID,
		Event:    event.Event,
		Data:     event.Data,
		Channel:  c.ChannelID,
		SocketID: ignore,
	})
}

func (a *app) Consume() error {
	c, err := a.Bus.Channel(a.AppID)
	if err != nil {
		log.Error(err)
		return err
	}

	defer close(c)
	for m := range c {
		go a.FindOrCreateChannelByChannelID(m.Channel).
			Publish(
				a,
				rawEvent{
					Event:   m.Event,
					Channel: m.Channel,
					Data:    m.Data,
				},
				m.SocketID,
			)

	}

	return nil
}

func (a *app) Unsubscribe(c *channel, conn *connection) error {
	return c.Unsubscribe(a, conn)
}

func (a *app) Subscribe(c *channel, conn *connection, data string) error {
	return c.Subscribe(a, conn, data)
}
