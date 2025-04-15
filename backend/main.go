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
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				fmt.Println("Error reading message:", err)
				break
			}
	
			if err := conn.WriteMessage(messageType, message); err != nil {
				fmt.Println("Error writing message:", err)
				break
			}
		}
	})

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
	
		conn, err := connectToClickHouse(&config)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Connection failed: %v", err)})
			return
		}
		defer conn.Close()
	
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

    router.POST("/api/query", func(c *gin.Context) {
		var queryRequest struct {
			Query    string          `json:"query" binding:"required"`
			Database ClickHouseConfig `json:"database"`
			Limit    int             `json:"limit"`
			Offset   int             `json:"offset"`
		}
	
		if err := c.ShouldBindJSON(&queryRequest); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
			return
		}
	
		conn, err := connectToClickHouse(&queryRequest.Database)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Connection failed: %v", err)})
			return
		}
		defer conn.Close()
	
		query := fmt.Sprintf("%s LIMIT %d OFFSET %d", queryRequest.Query, queryRequest.Limit, queryRequest.Offset)
		rows, err := conn.Query(context.Background(), query)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Query execution failed: %v", err)})
			return
		}
		defer rows.Close()
	
		results, err := processQueryResults(rows)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Processing results failed: %v", err)})
			return
		}
	
		c.JSON(http.StatusOK, results)
	})
	router.POST("/api/columns", func(c *gin.Context) {
		var request struct {
			Table    string `json:"table" binding:"required"`
			Database string `json:"database" binding:"required"`
			Host     string `json:"host" binding:"required"`
			Port     int    `json:"port" binding:"required"`
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}
	
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
			return
		}
	
		config := ClickHouseConfig{
			Host:     request.Host,
			Port:     request.Port,
			Database: request.Database,
			User:     request.Username,
			JWTToken: request.Password,
		}
	
		conn, err := connectToClickHouse(&config)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Connection failed: %v", err)})
			return
		}
		defer conn.Close()
	
		query := fmt.Sprintf("DESCRIBE TABLE %s", request.Table)
		rows, err := conn.Query(context.Background(), query)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to fetch columns: %v", err)})
			return
		}
		defer rows.Close()
	
		var columns []string
		for rows.Next() {
			var columnName string
			var columnType string
			var defaultType, defaultExpression, comment, codecExpression, ttlExpression string // placeholders for unused columns
			if err := rows.Scan(&columnName, &columnType, &defaultType, &defaultExpression, &comment, &codecExpression, &ttlExpression); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to read column name: %v", err)})
				return
			}
			columns = append(columns, columnName)
		}
	
		c.JSON(http.StatusOK, gin.H{"columns": columns})
	})
    if err := router.Run(":8080" ); err != nil {
        fmt.Printf("Error running router: %s\n", err)
    }
}

func processQueryResults(rows driver.Rows) (map[string]interface{}, error) {
	columnNames := rows.Columns()
	numColumns := len(columnNames)

	result := map[string]interface{}{
		"columns": columnNames,
		"rows":    [][]interface{}{},
	}

	var data [][]interface{}
	for rows.Next() {
		rowPtrs := make([]interface{}, numColumns)

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

		if err := rows.Scan(rowPtrs...); err != nil {
			return nil, fmt.Errorf("row scan error: %v", err)
		}

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

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %v", err)
	}

	result["rows"] = data
	result["rowCount"] = len(data)

	return result, nil
}

