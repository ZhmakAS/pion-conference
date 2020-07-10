package ws

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Client struct {
	id   string
	conn *websocket.Conn

	metadata string
	err      error
	mux      sync.RWMutex
}

func NewClientWithID(conn *websocket.Conn, id string) Client {
	if id == "" {
		id = uuid.New().String()
	}
	return Client{
		id:       id,
		conn:     conn,
		metadata: "ivan",
	}
}

func (c *Client) SetMetadata(metadata string) {
	c.metadata = metadata
}

func (c *Client) Metadata() string {
	return c.metadata
}

func (c *Client) Listen() <-chan Message {
	msgChan := make(chan Message)

	go func() {
		for {
			message, err := c.read()
			if err != nil {
				close(msgChan)
				c.err = err
				c.mux.Unlock()
				return
			}
			msgChan <- message
		}
	}()

	return msgChan
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) Err() error {
	c.mux.RLock()
	defer c.mux.RUnlock()
	return c.err
}

func (c *Client) SetTimeouts(timeout time.Time) error {
	var err error
	if err = c.conn.SetWriteDeadline(timeout); err != nil {
		return err
	}

	if err = c.conn.SetReadDeadline(timeout); err != nil {
		return err
	}

	return nil
}

func (c *Client) Write(msg Message) error {
	return c.conn.WriteJSON(msg)
}

func (c *Client) ID() string {
	return c.id
}

func (c *Client) read() (message Message, err error) {
	typ, data, err := c.conn.ReadMessage()
	if err != nil {
		err = fmt.Errorf("client.read - error reading data: %w", err)
		return
	}
	err = json.Unmarshal(data, &message)
	if err != nil {
		err = fmt.Errorf("client.read - error deserializing data: %w", err)
		return
	}
	if typ != websocket.TextMessage {
		err = fmt.Errorf("client.read - expected text message: %w", err)
	}
	return
}
