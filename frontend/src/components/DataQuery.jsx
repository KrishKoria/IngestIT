// frontend/src/components/DataQueryComponent.jsx
import React, { useState, useEffect, useRef } from "react";
import websocketService from "../services/websocketservice";

const DataQueryComponent = () => {
  const [query, setQuery] = useState("SELECT * FROM system.tables LIMIT 10");
  const [isStreaming, setIsStreaming] = useState(false);
  const [columns, setColumns] = useState([]);
  const [rows, setRows] = useState([]);
  const [status, setStatus] = useState("");
  const [error, setError] = useState("");
  const streamIdRef = useRef(null);

  useEffect(() => {
    // Connect to WebSocket when component mounts
    websocketService
      .connect()
      .catch((err) => setError(`WebSocket connection error: ${err.message}`));

    // Clean up when component unmounts
    return () => {
      if (streamIdRef.current) {
        websocketService.cancelQuery(streamIdRef.current);
      }
    };
  }, []);

  const handleExecuteQuery = async () => {
    // Reset previous results
    setColumns([]);
    setRows([]);
    setError("");
    setStatus("Executing query...");
    setIsStreaming(true);

    try {
      // Generate a new stream ID
      const streamId = crypto.randomUUID();
      streamIdRef.current = streamId;

      // Register handlers for this stream
      websocketService.onStream(streamId, {
        metadata: (msg) => {
          try {
            const metadata =
              typeof msg.data === "string" ? JSON.parse(msg.data) : msg.data;
            setColumns(metadata.columns);
          } catch (err) {
            console.error("Error parsing metadata:", err);
          }
        },
        data: (msg) => {
          try {
            const rowData =
              typeof msg.data === "string" ? JSON.parse(msg.data) : msg.data;
            setRows((prevRows) => [...prevRows, rowData]);
          } catch (err) {
            console.error("Error parsing row data:", err);
          }
        },
        complete: (msg) => {
          try {
            const result =
              typeof msg.data === "string" ? JSON.parse(msg.data) : msg.data;
            setStatus(
              `Query complete: ${result.rows} rows retrieved (${result.status})`
            );
            setIsStreaming(false);
            streamIdRef.current = null;
          } catch (err) {
            console.error("Error parsing complete message:", err);
          }
        },
        error: (msg) => {
          setError(msg.error);
          setStatus("Query failed");
          setIsStreaming(false);
          streamIdRef.current = null;
        },
      });

      // Execute the query
      await websocketService.executeQuery(query, streamId);
    } catch (err) {
      setError(`Failed to execute query: ${err.message}`);
      setStatus("Query failed");
      setIsStreaming(false);
    }
  };

  const handleCancelQuery = async () => {
    if (streamIdRef.current) {
      await websocketService.cancelQuery(streamIdRef.current);
      setStatus("Query cancelled");
      setIsStreaming(false);
    }
  };

  return (
    <div className="query-container">
      <h2>ClickHouse Query</h2>

      <div className="query-input">
        <textarea
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          rows={5}
          placeholder="Enter your SQL query here..."
          disabled={isStreaming}
        />
      </div>

      <div className="query-controls">
        {!isStreaming ? (
          <button onClick={handleExecuteQuery}>Execute Query</button>
        ) : (
          <button onClick={handleCancelQuery}>Cancel Query</button>
        )}
      </div>

      {status && <div className="query-status">{status}</div>}
      {error && <div className="query-error">{error}</div>}

      {columns.length > 0 && (
        <div className="query-results">
          <h3>Results {isStreaming && <span>(Streaming...)</span>}</h3>
          <div className="results-table-container">
            <table className="results-table">
              <thead>
                <tr>
                  {columns.map((col, idx) => (
                    <th key={idx}>{col}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {rows.map((row, rowIdx) => (
                  <tr key={rowIdx}>
                    {row.map((cell, cellIdx) => (
                      <td key={`${rowIdx}-${cellIdx}`}>
                        {cell === null ? "NULL" : String(cell)}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
};

export default DataQueryComponent;
