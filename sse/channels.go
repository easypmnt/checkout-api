package sse

import (
	"strings"

	"github.com/dustin/go-broadcast"
)

var channels = make(map[string]broadcast.Broadcaster)

func openListener(channelID string) chan interface{} {
	listener := make(chan interface{})
	channel(channelID).Register(listener)
	return listener
}

func closeListener(channelID string, listener chan interface{}) {
	channel(channelID).Unregister(listener)
	close(listener)
}

func deleteBroadcast(channelID string) {
	channelID = strings.ToLower(channelID)
	b, ok := channels[channelID]
	if ok {
		b.Close()
		delete(channels, channelID)
	}
}

func channel(channelID string) broadcast.Broadcaster {
	channelID = strings.ToLower(channelID)
	b, ok := channels[channelID]
	if !ok {
		b = broadcast.NewBroadcaster(10)
		channels[channelID] = b
	}
	return b
}
