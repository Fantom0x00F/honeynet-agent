package centralnodeconnection

import (
	"sync"
	"github.com/gorilla/websocket"
	"net/url"
	"time"
	"log"
	"errors"
	"bytes"
	"crypto/tls"
	"crypto/x509"
)

type CentralNodeConnection struct {
	secret         string
	responseSecret string
	url            url.URL
	tlsConfig      *tls.Config
	sync.Mutex
	*websocket.Conn
}

func NewCentralNodeConnection(url url.URL, rootPem []byte, secret string, response string) *CentralNodeConnection {
	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM(rootPem)
	if !ok {
		panic("failed to parse root certificate")
	}

	config := tls.Config{RootCAs: roots}

	return &CentralNodeConnection{
		secret:         secret,
		responseSecret: response,
		url:            url,
		tlsConfig:      &config,
	}
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
	dialer := websocket.Dialer{TLSClientConfig: connection.tlsConfig}
	newConn, _, err := dialer.Dial(connection.url.String(), nil)
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
	secret := []byte(connection.secret)

	connection.Conn.WriteMessage(websocket.TextMessage, secret)
	_, responce, err := connection.Conn.ReadMessage()
	if err != nil {
		return err
	}
	if !bytes.Equal(responce, []byte(connection.responseSecret)) {
		return errors.New("Secret's not matches!")
	}
	log.Println("Verification complete successfull!")
	return nil
}
