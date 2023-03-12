package main

import (
	"context"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

// Open a websocket connection and listen for events
func openWebsocketConnection(ctx context.Context, endpoint string, log *logrus.Entry, eg *errgroup.Group) *websocket.Conn {
	conn, _, err := websocket.DefaultDialer.Dial(endpoint, nil)
	if err != nil {
		log.WithError(err).Fatal("failed to connect to the websocket endpoint")
	}

	eg.Go(func() error {
		defer func() {
			log.Info("websocket connection listener stopped")
		}()

		<-ctx.Done()
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))

		wsCtx, wsCtxClose := context.WithTimeout(context.Background(), 5*time.Second)
		defer wsCtxClose()
		<-wsCtx.Done()

		return conn.Close()
	})

	conn.SetCloseHandler(func(code int, text string) error {
		log.WithFields(logrus.Fields{
			"code": code,
			"text": text,
		}).Info("websocket connection closed")
		return nil
	})

	return conn
}
