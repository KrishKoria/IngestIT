package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"net/http"
	"os"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/gin-gonic/gin"
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

func connectToClickHouse(config *ClickHouseConfig) (driver.Conn, error) {
	ctx := context.Background()
	// Create ClickHouse connection
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", config.Host, config.Port)},
		Auth: clickhouse.Auth{
			Database: config.Database,
			Username: config.User,
			Password: config.JWTToken,
		},
		TLS: &tls.Config{
			InsecureSkipVerify: true,
		}, // Add TLS configuration if needed
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to ClickHouse: %w", err)
	}

	// Test the connection
	if err := conn.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping ClickHouse: %w", err)
	}

	return conn, nil
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

	// Initialize Gin router
	router := gin.Default()

	// Example endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Server is running and connected to ClickHouse"})
	})

	// Start the HTTP server
	err = router.Run(":8080")
	if err != nil {
		fmt.Printf("Error running router: %s\n", err)
	}
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
