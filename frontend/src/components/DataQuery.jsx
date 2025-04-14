import React, { useState, useEffect, useRef } from "react";
import QueryInput from "./query/QueryInput";
import QueryControls from "./query/QueryControls";
import QueryStatus from "./query/QueryStatus";
import QueryResults from "./query/QueryResults";
import websocketService from "../services/websocketService";

const DataQueryComponent = () => {
  const [query, setQuery] = useState("SELECT * FROM system.tables LIMIT 10");
  const [isStreaming, setIsStreaming] = useState(false);
  const [columns, setColumns] = useState([]);
  const [rows, setRows] = useState([]);
  const [status, setStatus] = useState("");
  const [error, setError] = useState("");
  const streamIdRef = useRef(null);
  const [isLoadingData, setIsLoadingData] = useState(false);
  const timeoutRef = useRef(null);

  useEffect(() => {
    websocketService
      .connect()
      .catch((err) => setError(`WebSocket connection error: ${err.message}`));

    return () => {
      if (streamIdRef.current) {
        websocketService.cancelQuery(streamIdRef.current);
      }
    };
  }, []);

  const handleExecuteQuery = async () => {
    setColumns([]);
    setRows([]);
    setError("");
    setStatus("Executing query...");
    setIsStreaming(true);
    setIsLoadingData(true);

    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current);
    }

    timeoutRef.current = setTimeout(() => {
      if (isStreaming && rows.length === 0) {
        setError(
          "Query appears to be stalled. The data might be too large or there might be a connection issue."
        );
      }
    }, 10000);

    try {
      const streamId = crypto.randomUUID();
      streamIdRef.current = streamId;

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
            setIsLoadingData(false);
            const rowData =
              typeof msg.data === "string" ? JSON.parse(msg.data) : msg.data;
            setRows((prevRows) => [...prevRows, rowData]);
          } catch (err) {
            console.error("Error parsing row data:", err, msg);
            setError(`Error parsing row data: ${err.message}`);
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

      await websocketService.executeQuery(query, streamId);
    } catch (err) {
      setIsLoadingData(false);
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
      <QueryInput query={query} setQuery={setQuery} isStreaming={isStreaming} />
      <QueryControls
        isStreaming={isStreaming}
        onExecute={handleExecuteQuery}
        onCancel={handleCancelQuery}
      />
      <QueryStatus status={status} error={error} />
      {columns.length > 0 && (
        <QueryResults
          columns={columns}
          rows={rows}
          isStreaming={isStreaming}
          isLoadingData={isLoadingData}
        />
      )}
    </div>
  );
};

export default DataQueryComponent;
