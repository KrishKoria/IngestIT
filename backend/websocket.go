package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all connections (you might want to restrict this in production)
	},
}

// WebSocketHandler handles WebSocket connections
func WebSocketHandler(conn driver.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("Failed to set websocket upgrade: %v", err)
			return
		}
		defer ws.Close()

		// Create client connection
		client := &Client{
			conn:       ws,
			dbConn:     conn,
			send:       make(chan []byte, 256),
			ctx:        c.Request.Context(),
			cancelFunc: nil,
		}

		// Start client routines
		go client.writePump()
		client.readPump()
	}
}

// Client represents a WebSocket client
type Client struct {
	conn       *websocket.Conn
	dbConn     driver.Conn
	send       chan []byte
	ctx        context.Context
	cancelFunc context.CancelFunc
}

// Message represents the WebSocket message structure
type Message struct {
	Type     string          `json:"type"`
	Query    string          `json:"query,omitempty"`
	Data     json.RawMessage `json:"data,omitempty"`
	Error    string          `json:"error,omitempty"`
	StreamID string          `json:"streamId,omitempty"`
}

// readPump pumps messages from the WebSocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512 * 1024) // 512KB max message size
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		// Process the message
		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Error unmarshaling message: %v", err)
			c.sendError("Invalid message format")
			continue
		}

		// Handle different message types
		switch msg.Type {
		case "query":
			// Cancel any existing query
			if c.cancelFunc != nil {
				c.cancelFunc()
			}

			// Create a new context with cancellation
			ctx, cancel := context.WithCancel(c.ctx)
			c.cancelFunc = cancel

			// Execute the query in a goroutine
			go c.executeQuery(ctx, msg.Query, msg.StreamID)

		case "cancelQuery":
			if c.cancelFunc != nil {
				c.cancelFunc()
				c.cancelFunc = nil
			}

		default:
			c.sendError("Unknown message type")
		}
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
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

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// executeQuery executes a ClickHouse query and streams the results back to the client
func (c *Client) executeQuery(ctx context.Context, query string, streamID string) {
	// Execute the query
	rows, err := c.dbConn.Query(ctx, query)
	if err != nil {
		c.sendError(fmt.Sprintf("Query error: %v", err))
		return
	}
	defer rows.Close()
	columnNames := rows.Columns()
	numColumns := len(columnNames)

	// Send column metadata
	metadataMsg := Message{
		Type:     "metadata",
		StreamID: streamID,
		Data:     json.RawMessage(fmt.Sprintf(`{"columns":%q}`, columnNames)),
	}
	c.sendMessage(metadataMsg)

	// Stream results
	rowCount := 0
	for rows.Next() {
		// Create a slice to hold our row values
		row := make([]interface{}, numColumns)
		rowPtrs := make([]interface{}, numColumns)

		// Create pointers for each type based on column info
		for i := 0; i < numColumns; i++ {
			// Use appropriate type for each column - especially for numeric types
			columnTypes := rows.ColumnTypes()
			switch columnTypes[i].DatabaseTypeName() {
			case "UInt8", "UInt16", "UInt32":
				var val uint32
				rowPtrs[i] = &val
			case "UInt64":
				var val uint64
				rowPtrs[i] = &val
			case "Int8", "Int16", "Int32":
				var val int32
				rowPtrs[i] = &val
			case "Int64":
				var val int64
				rowPtrs[i] = &val
			case "Float32":
				var val float32
				rowPtrs[i] = &val
			case "Float64":
				var val float64
				rowPtrs[i] = &val
			case "String", "FixedString":
				var val string
				rowPtrs[i] = &val
			case "Date", "DateTime":
				var val time.Time
				rowPtrs[i] = &val
			default:
				var val interface{}
				rowPtrs[i] = &val
			}
		}

		// Scan the row into pointers
		if err := rows.Scan(rowPtrs...); err != nil {
			c.sendError(fmt.Sprintf("Row scan error: %v", err))
			return
		}

		// Extract values from pointers
		for i := 0; i < numColumns; i++ {
			switch v := rowPtrs[i].(type) {
			case *uint32:
				row[i] = *v
			case *uint64:
				row[i] = *v
			case *int32:
				row[i] = *v
			case *int64:
				row[i] = *v
			case *float32:
				row[i] = *v
			case *float64:
				row[i] = *v
			case *string:
				row[i] = *v
			case *time.Time:
				row[i] = *v
			case *interface{}:
				row[i] = *v
			}
		}

		// Convert row to JSON
		rowJSON, err := json.Marshal(row)
		if err != nil {
			c.sendError(fmt.Sprintf("JSON encoding error: %v", err))
			return
		}

		// Send row data
		dataMsg := Message{
			Type:     "data",
			StreamID: streamID,
			Data:     rowJSON,
		}
		c.sendMessage(dataMsg)

		rowCount++

		// Check if context was cancelled
		select {
		case <-ctx.Done():
			// Send completion message with information about cancellation
			completeMsg := Message{
				Type:     "complete",
				StreamID: streamID,
				Data:     json.RawMessage(fmt.Sprintf(`{"rows":%d,"status":"cancelled"}`, rowCount)),
			}
			c.sendMessage(completeMsg)
			return
		default:
			// Continue processing
		}
	}

	// Check for errors after iteration
	if err := rows.Err(); err != nil {
		c.sendError(fmt.Sprintf("Row iteration error: %v", err))
		return
	}

	// Send completion message
	completeMsg := Message{
		Type:     "complete",
		StreamID: streamID,
		Data:     json.RawMessage(fmt.Sprintf(`{"rows":%d,"status":"completed"}`, rowCount)),
	}
	c.sendMessage(completeMsg)
}

// sendError sends an error message to the client
func (c *Client) sendError(errorMsg string) {
	msg := Message{
		Type:  "error",
		Error: errorMsg,
	}
	c.sendMessage(msg)
}

// sendMessage sends a message to the client
func (c *Client) sendMessage(msg Message) {
	jsonData, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}
	c.send <- jsonData
}
