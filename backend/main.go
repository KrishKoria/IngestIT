package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/gin-gonic/gin"
	_ "github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

type ClickHouseConfig struct {
	Host     string
	Port     int
	Database string
	User     string
	JWTToken string
}

func loadConfig() (*ClickHouseConfig, error) {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		fmt.Println("Warning: No .env file found")
	}

	config := &ClickHouseConfig{
		Host:     os.Getenv("CLICKHOUSE_HOST"),
		Port:     getEnvAsInt("CLICKHOUSE_PORT", 0),
		Database: os.Getenv("CLICKHOUSE_DATABASE"),
		User:     os.Getenv("CLICKHOUSE_USER"),
		JWTToken: os.Getenv("CLICKHOUSE_JWT_TOKEN"),
	}

	// Validate required fields
	if config.Host == "" || config.Port == 0 || config.Database == "" || config.User == "" || config.JWTToken == "" {
		return nil, fmt.Errorf("missing required ClickHouse configuration")
	}

	return config, nil
}

func main() {
	// Load configuration
	config, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading configuration: %s\n", err)
		os.Exit(1)
	}

	// Connect to ClickHouse
	conn, err := connectToClickHouse(config)
	if err != nil {
		fmt.Printf("Error connecting to ClickHouse: %s\n", err)
		os.Exit(1)
	}
	defer func(conn driver.Conn) {
		err := conn.Close()
		if err != nil {
			fmt.Println("Failed to close connection")
		}
	}(conn)

	fmt.Println("Successfully connected to ClickHouse")

	// Initialize Gin router with CORS middleware
	router := gin.Default()
	router.Use(corsMiddleware())

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"message": "Server is running and connected to ClickHouse",
		})
	})

	// WebSocket endpoint for real-time data streaming
	router.GET("/ws", WebSocketHandler(conn))

	// API endpoint for executing queries (non-streaming)
	router.POST("/api/query", func(c *gin.Context) {
		var queryRequest struct {
			Query string `json:"query" binding:"required"`
		}

		if err := c.ShouldBindJSON(&queryRequest); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
			return
		}

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

	// File ingestion endpoint
	router.POST("/api/ingest", func(c *gin.Context) {
		// TODO: Implement file ingestion logic
		c.JSON(http.StatusNotImplemented, gin.H{"message": "File ingestion not yet implemented"})
	})

	// Start the HTTP server
	port := getEnvAsString("PORT", "8080")
	fmt.Printf("Server starting on port %s...\n", port)
	err = router.Run(":" + port)
	if err != nil {
		fmt.Printf("Error running router: %s\n", err)
	}
}

// CORS middleware
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
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

func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		var intValue int
		_, err := fmt.Sscanf(value, "%d", &intValue)
		if err != nil {
			return defaultValue
		}
		return intValue
	}
	return defaultValue
}

func getEnvAsString(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
