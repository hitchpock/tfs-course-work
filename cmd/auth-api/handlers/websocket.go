package handlers

import (
	"io"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"gitlab.com/hitchpock/tfs-course-work/internal/robot"
)

type WSClient struct {
	conn    *websocket.Conn
	robotID int
}

type WSClients struct {
	clients      map[int]*WSClient
	mutex        sync.Mutex
	nextID       int
	robotStorage robot.Storage
}

func NewWebsocket(robotStorage robot.Storage) *WSClients {
	ws := &WSClients{
		clients:      make(map[int]*WSClient),
		mutex:        sync.Mutex{},
		robotStorage: robotStorage,
	}

	return ws
}

func (c *WSClients) addClient(client *WSClient) int {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.nextID++
	c.clients[c.nextID] = client

	return c.nextID
}

func (c *WSClients) setRobot(clientID, robotID int) {
	c.mutex.Lock()
	c.clients[clientID].robotID = robotID
	c.mutex.Unlock()
}

func (c *WSClients) Broadcast(robotID int) {
	rob, err := c.robotStorage.FindByID(robotID)
	if err != nil {
		return
	}

	c.mutex.Lock()
	inactiveClients := make([]int, 0)

	for id, client := range c.clients {
		if client.robotID == robotID {
			if err := client.conn.WriteJSON(rob); err != nil {
				inactiveClients = append(inactiveClients, id)
			}
		}
	}

	c.mutex.Unlock()
	c.removeClients(inactiveClients...)
}

func (c *WSClients) removeClients(ids ...int) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for _, id := range ids {
		c.clients[id].conn.Close()
		delete(c.clients, id)
	}
}

func (c *WSClients) WSRobotDeltail(w http.ResponseWriter, r *http.Request) {
	var up websocket.Upgrader
	up.CheckOrigin = func(r *http.Request) bool {
		return true
	}

	conn, err := up.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	conn.SetPingHandler(func(appData string) error {
		err := conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(time.Second*1))
		if err == websocket.ErrCloseSent {
			return nil
		} else if e, ok := err.(net.Error); ok && e.Temporary() {
			return nil
		}
		return err
	})

	client := &WSClient{conn: conn}
	clientID := c.addClient(client)

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if err == io.EOF {
				c.removeClients(clientID)
				return
			}
		}

		robotID, err := strconv.Atoi(string(msg))
		if err == nil {
			c.setRobot(clientID, robotID)
			c.Broadcast(robotID)
		}
	}
}
