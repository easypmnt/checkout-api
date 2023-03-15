package sse

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

type (
	// SSE service struct
	Service struct {
		storage Storage
	}
)

// NewService factory func returns new SSE service
func NewService(storage Storage) *Service {
	return &Service{storage: storage}
}

// PubEvent func publishes data to channel with given ttl in seconds,
// returns error if any
func (s *Service) PubEvent(channelID string, data EventData, ttl int64) error {
	t := time.Now().UnixNano()
	event := Event{
		ID:        t,
		Data:      data,
		TTL:       ttl,
		Timestamp: t,
	}
	channel(channelID).Submit(event)
	return s.storeEvent(channelID, event)
}

// SubscribeToChannel func returns channel with events and history of events,
// returns error if any
func (s *Service) SubscribeToChannel(channelID, lastEventID string) (chan interface{}, []Event, error) {
	listener := openListener(channelID)
	history := make([]Event, 0, 50)
	var err error
	if lastEventID != "" {
		history, err = s.getEventsByLastID(channelID, lastEventID)
		if err != nil {
			return nil, nil, err
		}
		log.Printf("\nchannel: %s;\nlast event id: %s;\nhistory: %+v\n", channelID, lastEventID, history)
	}
	return listener, history, err
}

// Unsubscribe from channel with given listener
// returns error if any
func (s *Service) Unsubscribe(channelID string, listener chan interface{}) error {
	closeListener(channelID, listener)
	return nil
}

// DumpStorage func returns all events in channel
func (s *Service) DumpStorage(channelID string) []Event {
	channelID = strings.ToLower(channelID)
	return s.storage.GetAllInChannel(channelID)
}

func (s *Service) storeEvent(channelID string, event Event) error {
	channelID = strings.ToLower(channelID)
	return s.storage.Add(channelID, event)
}

func (s *Service) getEventsByLastID(channelID, lastEventID string) ([]Event, error) {
	channelID = strings.ToLower(channelID)
	var events []Event
	if lastEventID != "" && strings.Contains(lastEventID, ":") {
		parts := strings.Split(lastEventID, ":")
		if len(parts) != 2 {
			return nil, errors.New("wrong last event id")
		}
		sec, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf("convert last event id to int64: %v", err)
		}
		nsec, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("convert last event id to int64: %v", err)
		}
		lid := time.Unix(int64(sec), int64(nsec)).UnixNano()
		events = s.storage.GetByLastID(channelID, lid)
	}
	return events, nil
}
