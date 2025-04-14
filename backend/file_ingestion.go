package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

func exportClickHouseToFileWithProgress(conn driver.Conn, query, filePath string, progressChan chan string) error {
	ctx := context.Background()
	rows, err := conn.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Get column names
	columns := rows.Columns()
	if err := writer.Write(columns); err != nil {
		return fmt.Errorf("failed to write headers: %w", err)
	}

	rowCount := 0
	for rows.Next() {
		values := make([]interface{}, len(columns))
		for i := range values {
			var v interface{}
			values[i] = &v
		}

		// Scan the row into the values slice
		if err := rows.Scan(values...); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}

		// Convert all values to strings
		record := make([]string, len(columns))
		for i, value := range values {
			record[i] = fmt.Sprintf("%v", *(value.(*interface{})))
		}

		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write record: %w", err)
		}

		rowCount++
		if rowCount%100 == 0 {
			progressChan <- fmt.Sprintf("Exported %d rows", rowCount)
		}
	}

	progressChan <- fmt.Sprintf("Export completed with %d rows", rowCount)
	return nil
}

const batchSize = 1000

func importFileToClickHouseWithProgress(conn driver.Conn, filePath, tableName string, progressChan chan string) error {
	ctx := context.Background()

	// Open the file with a buffered reader
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	headers, err := reader.Read() // Read the header row
	if err != nil {
		return fmt.Errorf("failed to read file headers: %w", err)
	}

	// Prepare the batch for insertion once
	batch, err := conn.PrepareBatch(ctx, fmt.Sprintf("INSERT INTO %s (%s) VALUES", tableName, joinHeaders(headers)))
	if err != nil {
		return fmt.Errorf("failed to prepare batch: %w", err)
	}

	rowCount := 0
	for {
		record, err := reader.Read()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return fmt.Errorf("failed to read file: %w", err)
		}

		// Append the record to the batch
		values := make([]interface{}, len(record))
		for i, value := range record {
			values[i] = value
		}
		if err := batch.Append(values...); err != nil {
			return fmt.Errorf("failed to append record to batch: %w", err)
		}

		rowCount++
		// Send the batch when it reaches the batch size
		if rowCount%batchSize == 0 {
			if err := batch.Send(); err != nil {
				return fmt.Errorf("failed to send batch: %w", err)
			}
			progressChan <- fmt.Sprintf("Imported %d rows", rowCount)
		}
	}

	// Send any remaining records in the batch
	if err := batch.Send(); err != nil {
		return fmt.Errorf("failed to send final batch: %w", err)
	}

	progressChan <- fmt.Sprintf("Import completed with %d rows", rowCount)
	return nil
}

// Helper function to join headers for the SQL query
func joinHeaders(headers []string) string {
	return fmt.Sprintf("`%s`", joinStrings(headers, "`, `"))
}

// Helper function to join strings with a separator
func joinStrings(strings []string, sep string) string {
	result := ""
	for i, s := range strings {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
