package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type ClickHouseConfig struct {
    Host     string `json:"host"`
    Port     int    `json:"port"`
    Database string `json:"database"`
    User     string `json:"username"`
    JWTToken string `json:"password"`
}

func main() {

    // Initialize Gin router with CORS middleware
    router := gin.Default()
    router.Use(cors.Default())

	var upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true 
		},
	}

	router.GET("/ws", func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			fmt.Println("Failed to upgrade to WebSocket:", err)
			return
		}
		defer conn.Close()
	
		for {
			// Read message from client
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				fmt.Println("Error reading message:", err)
				break
			}
	
			// Echo message back to client
			if err := conn.WriteMessage(messageType, message); err != nil {
				fmt.Println("Error writing message:", err)
				break
			}
		}
	})

    // Health check endpoint
    router.GET("/health", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{
            "status":  "healthy",
            "message": "Server is running",
        })
    })

    router.POST("/api/connect", func(c *gin.Context) {
		var config ClickHouseConfig
		if err := c.ShouldBindJSON(&config); err != nil {
			fmt.Printf("Invalid connection parameters: %v\n", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid connection parameters"})
			return
		}
	
		fmt.Printf("Received connection parameters: %+v\n", config)
	
		// Attempt to connect to ClickHouse with the provided parameters
		conn, err := connectToClickHouse(&config)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Connection failed: %v", err)})
			return
		}
		defer conn.Close()
	
		// Fetch available tables as a test of the connection
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
	
		rows, err := conn.Query(ctx, "SHOW TABLES")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to fetch tables: %v", err)})
			return
		}
		defer rows.Close()
	
		var tables []string
		for rows.Next() {
			var tableName string
			if err := rows.Scan(&tableName); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to read table name: %v", err)})
				return
			}
			tables = append(tables, tableName)
		}
	
		c.JSON(http.StatusOK, gin.H{"tables": tables})
	})

    // API endpoint for executing queries (non-streaming)
    router.POST("/api/query", func(c *gin.Context) {
        var queryRequest struct {
            Query    string          `json:"query" binding:"required"`
            Database ClickHouseConfig `json:"database"`
        }

        if err := c.ShouldBindJSON(&queryRequest); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
            return
        }

        // Connect to ClickHouse with provided parameters
        conn, err := connectToClickHouse(&queryRequest.Database)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Connection failed: %v", err)})
            return
        }
        defer conn.Close()

        // Execute query with context
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()

        rows, err := conn.Query(ctx, queryRequest.Query)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Query execution failed: %v", err)})
            return
        }
        defer rows.Close()

        // Process results
        results, err := processQueryResults(rows)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Processing results failed: %v", err)})
            return
        }

        c.JSON(http.StatusOK, results)
    })
    if err := router.Run(":8080" ); err != nil {
        fmt.Printf("Error running router: %s\n", err)
    }
}

// processQueryResults converts ClickHouse query results to JSON-friendly format
func processQueryResults(rows driver.Rows) (map[string]interface{}, error) {
	// Get column information
	columnNames := rows.Columns()
	numColumns := len(columnNames)

	// Prepare result structure
	result := map[string]interface{}{
		"columns": columnNames,
		"rows":    [][]interface{}{},
	}

	// Collect rows
	var data [][]interface{}
	for rows.Next() {
		// Create a slice to hold our row values
		rowPtrs := make([]interface{}, numColumns)

		// Create pointers for each type based on column info
		for i := 0; i < numColumns; i++ {
			columnTypes := rows.ColumnTypes()
			switch columnTypes[i].DatabaseTypeName() {
			case "UInt8":
				var val uint8
				rowPtrs[i] = &val
			case "UInt16":
				var val uint16
				rowPtrs[i] = &val
			case "UInt32":
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
			case "String", "FixedString", "Nullable(String)":
				var val string
				rowPtrs[i] = &val
			case "Date", "DateTime":
				var val time.Time
				rowPtrs[i] = &val
			default:
				var val string
				rowPtrs[i] = &val
			}
		}

		// Scan the row into pointers
		if err := rows.Scan(rowPtrs...); err != nil {
			return nil, fmt.Errorf("row scan error: %v", err)
		}

		// Extract values from pointers
		rowValues := make([]interface{}, numColumns)
		for i := 0; i < numColumns; i++ {
			switch v := rowPtrs[i].(type) {
			case *uint32:
				rowValues[i] = *v
			case *uint64:
				rowValues[i] = *v
			case *int32:
				rowValues[i] = *v
			case *int64:
				rowValues[i] = *v
			case *float32:
				rowValues[i] = *v
			case *float64:
				rowValues[i] = *v
			case *string:
				rowValues[i] = *v
			case *time.Time:
				rowValues[i] = *v
			case *interface{}:
				rowValues[i] = *v
			}
		}

		data = append(data, rowValues)
	}

	// Check for errors after iteration
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %v", err)
	}

	result["rows"] = data
	result["rowCount"] = len(data)

	return result, nil
}

