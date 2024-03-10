package services

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gitlab.zixel.cn/go/framework"
	"google.golang.org/grpc/codes"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"messenger/configs"
	"messenger/restful/validations"
	"net/url"
	"strconv"
	"sync"
	"time"
)

// Server : Server is a struct which contains the required fields for our websocket server
type Server struct {
	ApplicationConnections           map[string]map[string]*ApplicationConnection
	ToMessengerConnectionInstances   map[string]*ToMessengerConnection
	FromMessengerConnectionInstances map[string]*FromMessengerConnection
	messengerPods                    *int
	clients                          map[string]*ClientConnection
	mu                               *sync.RWMutex
}

var ServerInstance *Server

// InitializeServer : creates a new GatewayServer type
func InitializeServer() *Server {
	Log.Info("Initializing Messenger server")
	// Initializing GatewayServer
	server := &Server{
		ApplicationConnections:           make(map[string]map[string]*ApplicationConnection),
		ToMessengerConnectionInstances:   make(map[string]*ToMessengerConnection),
		FromMessengerConnectionInstances: make(map[string]*FromMessengerConnection),
		messengerPods:                    new(int),
		mu:                               &sync.RWMutex{},
		clients:                          map[string]*ClientConnection{},
	}

	// Initializing messenger service connections
	initializeConnectionWithMessengerService(server)

	// Returning GatewayServer
	return server
}

// ApplicationConnection : ApplicationConnection is a struct which is used to store the connection to the session service
type ApplicationConnection struct {
	appId string
	conn  *websocket.Conn
	mu    *sync.Mutex
	send  chan []byte
}

// ToMessengerConnection : ApplicationConnection is a struct which is used to store the connection to the session service
type ToMessengerConnection struct {
	url  string
	conn *websocket.Conn
	mu   *sync.Mutex
	send chan []byte
}

// FromMessengerConnection : ApplicationConnection is a struct which is used to store the connection to the session service
type FromMessengerConnection struct {
	addr string
	conn *websocket.Conn
	mu   *sync.Mutex
	send chan []byte
}

// CreateWebSocketConnectionForApplication : CreateWebSocketConnectionForApplication calls the required functions to create a websocket connection
func CreateWebSocketConnectionForApplication(c *gin.Context) error {
	// Returning error and logging if failed to validate websocket connection
	if err := validations.WebSocketConnectionForApplication(c); err != nil {
		return err
	}

	// Returning error and logging if failed to create websocket connection
	if err := createWebSocketConnectionForApplication(c); err != nil {
		return err
	}

	// Returning nil if successful
	return nil
}

// createWebSocketConnectionForApplication : create websocket connection for users
func createWebSocketConnectionForApplication(c *gin.Context) error {
	ServerInstance.mu.Lock()
	defer ServerInstance.mu.Unlock()
	// Returning error and logging if failed to create websocket connection
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		Log.Error("Failed to create websocket connection: %+v", err)
		framework.Error(c, framework.NewServiceError(1000, ""))
		return CreateError(codes.Internal, 1000)
	}

	// Getting connection from WebSocketServer and setting connection
	connection := &ApplicationConnection{
		appId: c.Query("appId"),
		conn:  conn,
		send:  make(chan []byte, 256),
		mu:    &sync.Mutex{},
	}

	if ServerInstance.ApplicationConnections[c.Query("appId")] == nil {
		ServerInstance.ApplicationConnections[c.Query("appId")] = make(map[string]*ApplicationConnection)
		ServerInstance.ApplicationConnections[c.Query("appId")][conn.RemoteAddr().String()] = connection
	} else {
		ServerInstance.ApplicationConnections[c.Query("appId")][conn.RemoteAddr().String()] = connection
	}

	// Running connection reader and writer in goroutines
	go connection.writePump()
	go connection.readPump()

	// Returning nil if successful and send user success message
	message := gin.H{
		"content": "Application connected to messenger service",
	}
	jsonMessage, err := json.Marshal(message)
	if err != nil {
		Log.Error("Failed to marshal message: %+v", err)
		return err
	}
	connection.send <- jsonMessage
	return nil
}

// CreateWebSocketConnectionForInstance : CreateWebSocketConnectionForApplication calls the required functions to create a websocket connection
func CreateWebSocketConnectionForInstance(c *gin.Context) error {
	// Returning error and logging if failed to validate websocket connection
	if err := validations.WebSocketConnectionForInstance(c); err != nil {
		return err
	}

	// Returning error and logging if failed to create websocket connection
	if err := createWebSocketConnectionForInstance(c); err != nil {
		return err
	}

	// Returning nil if successful
	return nil
}

// createWebSocketConnectionForInstance : create websocket connection for users
func createWebSocketConnectionForInstance(c *gin.Context) error {
	ServerInstance.mu.Lock()
	defer ServerInstance.mu.Unlock()
	// Returning error and logging if failed to create websocket connection
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		Log.Error("Failed to create websocket connection: %+v", err)
		framework.Error(c, framework.NewServiceError(1000, ""))
		return CreateError(codes.Internal, 1000)
	}

	// Getting connection from WebSocketServer and setting connection
	connection := &FromMessengerConnection{
		addr: conn.RemoteAddr().String(),
		conn: conn,
		send: make(chan []byte, 256),
		mu:   &sync.Mutex{},
	}

	ServerInstance.FromMessengerConnectionInstances[conn.RemoteAddr().String()] = connection

	// Running connection reader and writer in goroutines
	go connection.writePump()
	go connection.readPump()

	// Returning nil if successful
	return nil
}

// readPump : read messages from client
func (conn *ApplicationConnection) readPump() {
	// Deferring closing of connection
	defer func() {
		conn.disconnect()
	}()
	// Setting connection read deadline and read limit
	conn.conn.SetReadLimit(maxMessageSize)
	err := conn.conn.SetReadDeadline(time.Now().Add(pongWait))

	// Logging error if failed to set read deadline
	if err != nil {
		Log.Error("Failed to set read deadline: %+v", err)
		return
	}

	// Logging error if failed to set pong handler
	conn.conn.SetPongHandler(func(string) error {
		err := conn.conn.SetReadDeadline(time.Now().Add(pongWait))
		if err != nil {
			return err
		}
		return nil
	})

	// Start endless read loop, waiting for messages from conn
	for {
		// Reading message from conn
		_, _, err := conn.conn.ReadMessage()

		// Logging error if failed to read message and breaking loop
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				Log.Error("Failed to read message: %+v", err)
			}
			break
		}
	}
}

// writePump : write messages to client
func (conn *ApplicationConnection) writePump() {
	// Initializing ticker
	ticker := time.NewTicker(pingPeriod)

	// Deferring closing of connection and ticker
	defer func() {
		ticker.Stop()
		err := conn.conn.Close()
		if err != nil {
			return
		}
	}()

	// Start endless write loop, waiting for messages from conn
	for {
		select {
		// Waiting for send channel message from conn
		case message, ok := <-conn.send:
			func() {
				conn.mu.Lock()
				defer conn.mu.Unlock()
				// Logging error if failed to set write deadline and breaking loop
				err := conn.conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err != nil {
					Log.Error("Failed to set write deadline: %+v", err)
					return
				}
				// Logging error if failed to write message and breaking loop
				if !ok {
					// The WsServer closed the channel.
					err := conn.conn.WriteMessage(websocket.CloseMessage, []byte{})
					if err != nil {
						Log.Error("Failed to write message: %+v", err)
						return
					}
				}

				// Logging error if failed get io.writeClosure
				w, err := conn.conn.NextWriter(websocket.TextMessage)
				if err != nil {
					Log.Error("Failed to get io.writeClosure: %+v", err)
					return
				}

				// Logging error if failed to write message
				_, err = w.Write(message)
				if err != nil {
					Log.Error("Failed to write message: %+v", err)
					return
				}

				// Attach queued chat messages to the current websocket message.
				n := len(conn.send)

				// Iterating through all queued messages
				for i := 0; i < n; i++ {
					// Logging error if failed to add newline to message
					_, err := w.Write(newline)
					if err != nil {
						Log.Error("Failed to add newline to message: %+v", err)
						return
					}

					// Logging error if failed to get message from send channel and writing message
					_, err = w.Write(<-conn.send)
					if err != nil {
						Log.Error("Failed to get message from send channel and write message: %+v", err)
						return
					}
				}

				// Logging error if failed to close writer
				if err := w.Close(); err != nil {
					Log.Error("Failed to close writer: %+v", err)
					return
				}
			}()
		// Calling this case when ticker ticks
		case <-ticker.C:
			// Logging error if failed to set write deadline and breaking loop
			err := conn.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err != nil {
				Log.Error("Failed to set write deadline: %+v", err)
				return
			}

			// Logging error if failed to write ping message and breaking loop
			if err := conn.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				Log.Error("Failed to write ping message: %+v", err)
				return
			}
		}
	}
}

// disconnect : disconnect client from server
func (conn *ApplicationConnection) disconnect() {
	conn.mu.Lock()
	defer conn.mu.Unlock()
	ServerInstance.mu.Lock()
	defer ServerInstance.mu.Unlock()

	// Logging if failed to close conn send channel and connection
	err := conn.conn.Close()
	if err != nil {
		Log.Error("Failed to close conn send channel and connection: %+v", err)
		return
	}

	// Deleting connection from connections map
	delete(ServerInstance.ApplicationConnections[conn.appId], conn.conn.RemoteAddr().String())

	Log.Info(conn.conn.RemoteAddr().String())
}

// readPump : read messages from client
func (conn *ToMessengerConnection) readPump() {
	// Deferring closing of connection
	defer func() {
		conn.disconnect()
	}()
	// Setting connection read deadline and read limit
	conn.conn.SetReadLimit(maxMessageSize)
	err := conn.conn.SetReadDeadline(time.Now().Add(pongWait))

	// Logging error if failed to set read deadline
	if err != nil {
		Log.Error("Failed to set read deadline: %+v", err)
		return
	}

	// Logging error if failed to set pong handler
	conn.conn.SetPongHandler(func(string) error {
		err := conn.conn.SetReadDeadline(time.Now().Add(pongWait))
		if err != nil {
			return err
		}
		return nil
	})

	// Start endless read loop, waiting for messages from conn
	for {
		// Reading message from conn
		_, _, err := conn.conn.ReadMessage()

		// Logging error if failed to read message and breaking loop
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				Log.Error("Failed to read message: %+v", err)
			}
			break
		}
	}
}

// writePump : write messages to client
func (conn *ToMessengerConnection) writePump() {
	// Initializing ticker
	ticker := time.NewTicker(pingPeriod)

	// Deferring closing of connection and ticker
	defer func() {
		ticker.Stop()
		err := conn.conn.Close()
		if err != nil {
			return
		}
	}()

	// Start endless write loop, waiting for messages from conn
	for {
		select {
		// Waiting for send channel message from conn
		case message, ok := <-conn.send:
			func() {
				conn.mu.Lock()
				defer conn.mu.Unlock()
				// Logging error if failed to set write deadline and breaking loop
				err := conn.conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err != nil {
					Log.Error("Failed to set write deadline: %+v", err)
					return
				}
				// Logging error if failed to write message and breaking loop
				if !ok {
					// The WsServer closed the channel.
					err := conn.conn.WriteMessage(websocket.CloseMessage, []byte{})
					if err != nil {
						Log.Error("Failed to write message: %+v", err)
						return
					}
				}

				// Logging error if failed get io.writeClosure
				w, err := conn.conn.NextWriter(websocket.TextMessage)
				if err != nil {
					Log.Error("Failed to get io.writeClosure: %+v", err)
					return
				}

				// Logging error if failed to write message
				_, err = w.Write(message)
				if err != nil {
					Log.Error("Failed to write message: %+v", err)
					return
				}

				// Attach queued chat messages to the current websocket message.
				n := len(conn.send)

				// Iterating through all queued messages
				for i := 0; i < n; i++ {
					// Logging error if failed to add newline to message
					_, err := w.Write(newline)
					if err != nil {
						Log.Error("Failed to add newline to message: %+v", err)
						return
					}

					// Logging error if failed to get message from send channel and writing message
					_, err = w.Write(<-conn.send)
					if err != nil {
						Log.Error("Failed to get message from send channel and write message: %+v", err)
						return
					}
				}

				// Logging error if failed to close writer
				if err := w.Close(); err != nil {
					Log.Error("Failed to close writer: %+v", err)
					return
				}
			}()

		// Calling this case when ticker ticks
		case <-ticker.C:
			// Logging error if failed to set write deadline and breaking loop
			err := conn.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err != nil {
				Log.Error("Failed to set write deadline: %+v", err)
				return
			}

			// Logging error if failed to write ping message and breaking loop
			if err := conn.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				Log.Error("Failed to write ping message: %+v", err)
				return
			}
		}
	}
}

// disconnect : disconnect client from server
func (conn *ToMessengerConnection) disconnect() {
	conn.mu.Lock()
	defer conn.mu.Unlock()
	ServerInstance.mu.Lock()
	defer ServerInstance.mu.Unlock()
	// Logging if failed to close conn send channel and connection
	err := conn.conn.Close()
	if err != nil {
		Log.Error("Failed to close conn send channel and connection: %+v", err)
		return
	}

	// Deleting connection from connections map
	delete(ServerInstance.ToMessengerConnectionInstances, conn.url)

	Log.Info(conn.conn.RemoteAddr().String())
}

// readPump : read messages from client
func (conn *FromMessengerConnection) readPump() {
	// Deferring closing of connection
	defer func() {
		conn.disconnect()
	}()
	// Setting connection read deadline and read limit
	conn.conn.SetReadLimit(maxMessageSize)
	err := conn.conn.SetReadDeadline(time.Now().Add(pongWait))

	// Logging error if failed to set read deadline
	if err != nil {
		Log.Error("Failed to set read deadline: %+v", err)
		return
	}

	// Logging error if failed to set pong handler
	conn.conn.SetPongHandler(func(string) error {
		err := conn.conn.SetReadDeadline(time.Now().Add(pongWait))
		if err != nil {
			return err
		}
		return nil
	})

	// Start endless read loop, waiting for messages from conn
	for {
		// Reading message from conn
		_, jsonMessage, err := conn.conn.ReadMessage()

		// Logging error if failed to read message and breaking loop
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				Log.Error("Failed to read message: %+v", err)
			}
			break
		}

		// Handling message from messengerConn
		conn.handleBroadcastNotification(jsonMessage)
	}
}

// writePump : write messages to client
func (conn *FromMessengerConnection) writePump() {
	// Initializing ticker
	ticker := time.NewTicker(pingPeriod)

	// Deferring closing of connection and ticker
	defer func() {
		ticker.Stop()
		err := conn.conn.Close()
		if err != nil {
			return
		}
	}()

	// Start endless write loop, waiting for messages from conn
	for {
		select {
		// Waiting for send channel message from conn
		case message, ok := <-conn.send:
			func() {
				conn.mu.Lock()
				defer conn.mu.Unlock()
				// Logging error if failed to set write deadline and breaking loop
				err := conn.conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err != nil {
					Log.Error("Failed to set write deadline: %+v", err)
					return
				}
				// Logging error if failed to write message and breaking loop
				if !ok {
					// The WsServer closed the channel.
					err := conn.conn.WriteMessage(websocket.CloseMessage, []byte{})
					if err != nil {
						Log.Error("Failed to write message: %+v", err)
						return
					}
				}

				// Logging error if failed get io.writeClosure
				w, err := conn.conn.NextWriter(websocket.TextMessage)
				if err != nil {
					Log.Error("Failed to get io.writeClosure: %+v", err)
					return
				}

				// Logging error if failed to write message
				_, err = w.Write(message)
				if err != nil {
					Log.Error("Failed to write message: %+v", err)
					return
				}

				// Attach queued chat messages to the current websocket message.
				n := len(conn.send)

				// Iterating through all queued messages
				for i := 0; i < n; i++ {
					// Logging error if failed to add newline to message
					_, err := w.Write(newline)
					if err != nil {
						Log.Error("Failed to add newline to message: %+v", err)
						return
					}

					// Logging error if failed to get message from send channel and writing message
					_, err = w.Write(<-conn.send)
					if err != nil {
						Log.Error("Failed to get message from send channel and write message: %+v", err)
						return
					}
				}

				// Logging error if failed to close writer
				if err := w.Close(); err != nil {
					Log.Error("Failed to close writer: %+v", err)
					return
				}
			}()

		// Calling this case when ticker ticks
		case <-ticker.C:
			// Logging error if failed to set write deadline and breaking loop
			err := conn.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err != nil {
				Log.Error("Failed to set write deadline: %+v", err)
				return
			}

			// Logging error if failed to write ping message and breaking loop
			if err := conn.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				Log.Error("Failed to write ping message: %+v", err)
				return
			}
		}
	}
}

// disconnect : disconnect client from server
func (conn *FromMessengerConnection) disconnect() {
	conn.mu.Lock()
	defer conn.mu.Unlock()
	ServerInstance.mu.Lock()
	defer ServerInstance.mu.Unlock()
	// Logging if failed to close conn send channel and connection
	err := conn.conn.Close()
	if err != nil {
		Log.Error("Failed to close conn send channel and connection: %+v", err)
		return
	}

	// Deleting connection from connections map
	delete(ServerInstance.FromMessengerConnectionInstances, conn.addr)

	Log.Info(conn.conn.RemoteAddr().String())
}

// handleBroadcastNotification : handle message from client based on message action
func (conn *FromMessengerConnection) handleBroadcastNotification(jsonMessage []byte) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ServerInstance.mu.Lock()
	defer ServerInstance.mu.Unlock()
	// Initializing message
	var message *Message

	// Unmarshalling message from client
	if err := json.Unmarshal(jsonMessage, &message); err != nil {
		Log.Error(err.Error())
		return
	}

	var configuration Configuration

	// Returning error and Logging if failed to find message configuration
	if err := getConfigurationByClassId(ctx, message.ClassId, &configuration); err != nil {
		Log.Error(err.Error())
		return
	}

	// Sending message to application if application connection is available
	if applicationConnections, ok := ServerInstance.ApplicationConnections[message.ClassId]; ok {
		for _, connection := range applicationConnections {
			connection.send <- message.Encode()
		}
	}

	// Iterating through all openIds in message target and adding message to cache
	for _, receiver := range message.Target {
		// Sending message to user if user connection is available
		sendMessageToUserSocket(receiver, message, &configuration, true)
	}
}

// initializeConnectionWithMessenger : initialize connection with messenger instance
func initializeConnectionWithMessengerService(server *Server) {
	// Creating WebSocket connection to session service based on environment
	if configs.IsReleaseMode() {
		Log.Info("Start to connect to messenger service")

		// Logging error if failed to create websocket connection to session service on Kubernetes pods
		if err := createWebSocketConnectionsWithMessengerServicePods(server); err != nil {
			Log.Error()
		}
	}

	// Running a goroutine which handles reconnection of websocket sessionServiceConnections to session service
	go func() {
		// Initializing ticker
		ticker := time.NewTicker(10 * time.Second)

		// Start endless loop
		for {
			select {
			case <-ticker.C:
				if configs.IsReleaseMode() {
					service, err := manager.GetService(configs.Namespace, "messenger-svc")
					if err != nil {
						Log.Error(err.Error())
						return
					}
					selector := service.Spec.Selector
					ports := service.Spec.Ports
					set := labels.Set(selector)

					pods, err := manager.ListPods(configs.Namespace, metaV1.ListOptions{
						LabelSelector: set.AsSelector().String(),
					})
					if err != nil {
						Log.Error(err.Error())
						return
					}

					// Iterating through all pod in namespace
					var check bool
					for _, pod := range pods.Items {
						check = func() bool {
							server.mu.RLock()
							defer server.mu.RUnlock()
							if pod.Status.Phase == "Running" && pod.Name != configs.Hostname {
								u := url.URL{
									Scheme: "ws",
									Host:   pod.Status.PodIP + ":" + fmt.Sprint(ports[0].TargetPort.IntVal),
									Path:   "/messenger/server/instance/ws",
								}
								if _, ok := server.ToMessengerConnectionInstances[u.String()]; !ok {
									Log.Info("Found new messenger service pod: " + u.String() + " and reconnecting to it")
									return true
								} else {
									return false
								}
							} else {
								return false
							}
						}()
						if check {
							break
						}
					}
					if check {
						Log.Info("Reconnecting to messenger service instances")

						// Logging error if failed to create websocket connection to session service on Kubernetes pods
						if err := createWebSocketConnectionsWithMessengerServicePods(server); err != nil {
							Log.Error()
						}
					}
				}
			}
		}
	}()
}

// createWebSocketConnectionsWithMessengerServicePods creates websocket connections to messenger service on Kubernetes pods
func createWebSocketConnectionsWithMessengerServicePods(server *Server) error {
	for _, connection := range server.ToMessengerConnectionInstances {
		connection.disconnect()
	}

	server.mu.Lock()
	defer server.mu.Unlock()
	server.ToMessengerConnectionInstances = make(map[string]*ToMessengerConnection)
	server.messengerPods = new(int)

	service, err := manager.GetService(configs.Namespace, "messenger-svc")
	if err != nil {
		Log.Error(err.Error())
		return err
	}

	selector := service.Spec.Selector
	ports := service.Spec.Ports
	set := labels.Set(selector)
	pods, err := manager.ListPods(configs.Namespace, metaV1.ListOptions{
		LabelSelector: set.AsSelector().String(),
	})
	if err != nil {
		Log.Error(err.Error())
		return err
	}

	Log.Info("Found messenger service pods: " + fmt.Sprint(len(pods.Items)))
	var count int
	// Iterating through all pod in namespace
	for _, pod := range pods.Items {
		// Checking if pod is running
		if pod.Status.Phase == "Running" && pod.Name != configs.Hostname {
			Log.Info("Found messenger service pod: " + pod.Name)
			count++
			// Creating Request URL
			u := url.URL{
				Scheme: "ws",
				Host:   pod.Status.PodIP + ":" + fmt.Sprint(ports[0].TargetPort.IntVal),
				Path:   "/messenger/server/instance/ws",
			}
			// Creating new websocket connection with pod
			conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)

			// Logging error if failed to create websocket connection
			if err != nil {
				Log.Error(err.Error())
				continue
			} else {
				// Initializing SessionServiceConnection
				connection := &ToMessengerConnection{
					url:  u.String(),
					conn: conn,
					send: make(chan []byte, 256),
					mu:   &sync.Mutex{},
				}

				// Appending connection to server sessionServiceConnections
				server.ToMessengerConnectionInstances[u.String()] = connection

				// Running readPump and writePump for connection
				go connection.readPump()
				go connection.writePump()

				// Logging if connection is established
				Log.Info("Connected to messenger Service instance: " + u.String())
			}
		}
	}
	*server.messengerPods = count
	Log.Info("Total messenger pod: " + strconv.Itoa(*server.messengerPods))
	return nil
}
