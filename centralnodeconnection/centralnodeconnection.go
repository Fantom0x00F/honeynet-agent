package centralnodeconnection

import (
	"sync"
	"github.com/gorilla/websocket"
	"net/url"
	"time"
	"log"
	"errors"
	"bytes"
)

type CentralNodeConnection struct {
	Secret         string
	ResponseSecret string
	Url            url.URL
	sync.Mutex
	*websocket.Conn
}

func (connection *CentralNodeConnection) Open() error {
	return connection.tryReconnect()
}

func (connection *CentralNodeConnection) ReadMessage() (messageType int, p []byte, err error) {
	if connection.Conn == nil {
		connection.Lock()
		connection.Unlock()
	}
	if connection.Conn == nil {
		return websocket.CloseAbnormalClosure, nil, errors.New("Connection failed")
	}
	return connection.Conn.ReadMessage()
}

func (connection *CentralNodeConnection) WriteMessage(messageType int, data []byte) error {
	if connection.Conn == nil {
		connection.Lock()
		connection.Unlock()
	}
	if connection.Conn == nil {
		return errors.New("Connection failed")
	}
	return connection.Conn.WriteMessage(messageType, data)
}

func (connection *CentralNodeConnection) Reconnect() error {
	log.Println("We try to reconnect")
	connection.Lock()
	defer connection.Unlock()
	if connection.Conn != nil {
		err := connection.Conn.Close()
		if err != nil {
			log.Println("Error on close socket", err)
		}
		connection.Conn = nil
	}
	ticker := time.NewTicker(time.Second)
	delay := time.After(time.Minute)
	for {
		select {
		case <-ticker.C:
			log.Println("try to reconnect")
			if err := connection.tryReconnect(); err != nil {
				log.Println("Fail ", err)
			} else {
				return nil
			}
		case <-delay:
			log.Println("Failed to connect")
			return errors.New("Failed to connect")
		}
	}
	return nil
}

func (connection *CentralNodeConnection) tryReconnect() error {
	newConn, _, err := websocket.DefaultDialer.Dial(connection.Url.String(), nil)
	if err != nil {
		connection.Conn = nil
		return err
	}
	connection.Conn = newConn
	if err := connection.verifyOnConnectionStart(); err != nil {
		connection.Conn = nil
		return err
	}
	return nil
}

func (connection *CentralNodeConnection) verifyOnConnectionStart() error {
	secret := []byte(connection.Secret)

	connection.Conn.WriteMessage(websocket.TextMessage, secret)
	_, responce, err := connection.Conn.ReadMessage()
	if err != nil {
		return err
	}
	if !bytes.Equal(responce, []byte(connection.ResponseSecret)) {
		return errors.New("Secret's not matches!")
	}
	log.Println("Verification complete successfull!")
	return nil
}