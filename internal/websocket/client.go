package websocket

import (
    "encoding/json"
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/gorilla/websocket"
    "github.com/sirupsen/logrus"
    "github.com/google/uuid"
)

const (
    // Time allowed to write a message to the peer
    writeWait = 10 * time.Second

    // Time allowed to read the next pong message from the peer
    pongWait = 60 * time.Second

    // Send pings to peer with this period. Must be less than pongWait
    pingPeriod = (pongWait * 9) / 10

    // Maximum message size allowed from peer
    maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool {
        // Allow all origins for now (configure based on your needs)
        return true
    },
}

// Client is a middleman between the websocket connection and the hub
type Client struct {
    // Unique client identifier
    ID string

    // The websocket connection
    conn *websocket.Conn

    // Buffered channel of outbound messages
    send chan []byte

    // Hub reference
    hub *Hub

    // Logger
    logger *logrus.Logger

    // Client metadata
    UserAgent  string    `json:"user_agent"`
    RemoteAddr string    `json:"remote_addr"`
    ConnectedAt time.Time `json:"connected_at"`

    // Room subscriptions
    rooms map[int]bool
}

// HandleWebSocket handles websocket requests from clients
func HandleWebSocket(hub *Hub, w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        hub.logger.WithError(err).Error("Failed to upgrade WebSocket connection")
        return
    }

    client := &Client{
        ID:          uuid.New().String(),
        conn:        conn,
        send:        make(chan []byte, 256),
        hub:         hub,
        logger:      hub.logger,
        UserAgent:   r.Header.Get("User-Agent"),
        RemoteAddr:  r.RemoteAddr,
        ConnectedAt: time.Now(),
        rooms:       make(map[int]bool),
    }

    // Register the client with the hub
    client.hub.register <- client

    // Allow collection of memory referenced by the caller by doing all work in
    // new goroutines
    go client.writePump()
    go client.readPump()
}

// HandleWebSocketGin is a Gin-compatible wrapper for HandleWebSocket
func HandleWebSocketGin(hub *Hub) gin.HandlerFunc {
    return func(c *gin.Context) {
        HandleWebSocket(hub, c.Writer, c.Request)
    }
}

// readPump pumps messages from the websocket connection to the hub
func (c *Client) readPump() {
    defer func() {
        c.hub.unregister <- c
        c.conn.Close()
    }()

    c.conn.SetReadLimit(maxMessageSize)
    c.conn.SetReadDeadline(time.Now().Add(pongWait))
    c.conn.SetPongHandler(func(string) error {
        c.conn.SetReadDeadline(time.Now().Add(pongWait))
        return nil
    })

    for {
        _, message, err := c.conn.ReadMessage()
        if err != nil {
            if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
                c.logger.WithError(err).Error("WebSocket connection error")
            }
            break
        }

        // Handle incoming message
        c.handleMessage(message)
        c.hub.stats.MessagesReceived++
    }
}

// writePump pumps messages from the hub to the websocket connection
func (c *Client) writePump() {
    ticker := time.NewTicker(pingPeriod)
    defer func() {
        ticker.Stop()
        c.conn.Close()
    }()

    for {
        select {
        case message, ok := <-c.send:
            c.conn.SetWriteDeadline(time.Now().Add(writeWait))
            if !ok {
                // The hub closed the channel
                c.conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }

            w, err := c.conn.NextWriter(websocket.TextMessage)
            if err != nil {
                return
            }
            w.Write(message)

            // Add queued messages to the current websocket message
            n := len(c.send)
            for i := 0; i < n; i++ {
                w.Write([]byte{'\n'})
                w.Write(<-c.send)
            }

            if err := w.Close(); err != nil {
                return
            }

        case <-ticker.C:
            c.conn.SetWriteDeadline(time.Now().Add(writeWait))
            if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                return
            }
        }
    }
}

// handleMessage processes incoming messages from the client
func (c *Client) handleMessage(message []byte) {
    var msg Message
    if err := json.Unmarshal(message, &msg); err != nil {
        c.logger.WithError(err).Error("Failed to unmarshal WebSocket message")
        return
    }

    switch msg.Type {
    case "subscribe_room":
        if roomID, ok := msg.Data["room_id"].(float64); ok {
            c.SubscribeToRoom(int(roomID))
        }
    case "unsubscribe_room":
        if roomID, ok := msg.Data["room_id"].(float64); ok {
            c.UnsubscribeFromRoom(int(roomID))
        }
    case "ping":
        // Respond with pong
        pong := Message{
            Type: "pong",
            Data: map[string]interface{}{
                "timestamp": time.Now().UTC(),
            },
        }
        c.send <- pong.ToJSON()
    default:
        c.logger.WithField("message_type", msg.Type).Warn("Unknown WebSocket message type")
    }
}

// SubscribeToRoom subscribes the client to room updates
func (c *Client) SubscribeToRoom(roomID int) {
    c.rooms[roomID] = true
    c.logger.WithFields(logrus.Fields{
        "client_id": c.ID,
        "room_id":   roomID,
    }).Info("Client subscribed to room")
}

// UnsubscribeFromRoom unsubscribes the client from room updates
func (c *Client) UnsubscribeFromRoom(roomID int) {
    delete(c.rooms, roomID)
    c.logger.WithFields(logrus.Fields{
        "client_id": c.ID,
        "room_id":   roomID,
    }).Info("Client unsubscribed from room")
}

// IsInRoom checks if the client is subscribed to a room
func (c *Client) IsInRoom(roomID int) bool {
    return c.rooms[roomID]
} 