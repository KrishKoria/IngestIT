import React, { useState, useRef } from "react";
import websocketService from "../services/websocketService.js";
import SourceTargetSelector from "./selectors/SourceTargetSelectors";
import ConnectionParameters from "./selectors/ConnectionParameters";
import TableList from "./misc/TableList";
import ColumnSelector from "./selectors/ColumnSelector";
import ActionButtons from "./misc/ActionButtons";
import StatusDisplay from "./misc/StatusDisplay";
import ResultDisplay from "./misc/ResultDisplay";
import QueryControls from "./query/QueryControls";
import QueryStatus from "./query/QueryStatus";
import QueryResults from "./query/QueryResults";

const DataQueryComponent = () => {
  // State for query execution
  const [isStreaming, setIsStreaming] = useState(false);
  const [columns, setColumns] = useState([]);
  const [rows, setRows] = useState([]);
  const [status, setStatus] = useState("");
  const [error, setError] = useState("");
  const [isLoadingData, setIsLoadingData] = useState(false);
  const [currentPage, setCurrentPage] = useState(1);
  const [rowsPerPage] = useState(100);

  // State for new UI requirements
  const [sourceType, setSourceType] = useState("clickhouse");
  const [targetType, setTargetType] = useState("clickhouse");
  const [parameters, setParameters] = useState({});
  const [tables, setTables] = useState([]);
  const [selectedTable, setSelectedTable] = useState("");
  const [selectedColumns, setSelectedColumns] = useState([]);
  const [result, setResult] = useState("");

  const streamIdRef = useRef(null);

  const handleConnect = async () => {
    setStatus("Connecting...");
    setError("");

    try {
      const response = await fetch("http://localhost:8080/api/connect", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          host: parameters.host,
          port: parseInt(parameters.port, 10),
          database: parameters.database,
          username: parameters.username,
          password: parameters.password,
        }),
      });

      if (!response.ok) {
        const errorData = await response.json();
        setError(`Connection failed: ${errorData.error}`);
        setStatus("Connection failed");
        return;
      }

      const data = await response.json();
      setTables(data.tables);
      setStatus("Connected");
    } catch (err) {
      setError(`Connection error: ${err.message}`);
      setStatus("Connection failed");
    }
  };

  const handleExecuteQuery = async () => {
    if (!selectedTable) {
      setError("Please select a table first.");
      return;
    }

    if (selectedColumns.length === 0) {
      setError("Please select at least one column.");
      return;
    }

    setColumns([]);
    setRows([]);
    setError("");
    setStatus("Executing query...");
    setIsStreaming(true);
    setIsLoadingData(true);

    try {
      console.log("Ensuring WebSocket connection...");
      await websocketService.ensureConnected();

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

      const offset = (currentPage - 1) * rowsPerPage;
      const query = `SELECT ${selectedColumns.join(
        ", "
      )} FROM ${selectedTable} LIMIT ${rowsPerPage} OFFSET ${offset}`;
      console.log("Executing query:", query);
      await websocketService.executeQuery(query, streamId);
    } catch (err) {
      setIsLoadingData(false);
      setError(`Failed to execute query: ${err.message}`);
      setStatus("Query failed");
      setIsStreaming(false);
    }
  };

  const handleNextPage = () => {
    setCurrentPage((prev) => prev + 1);
    handleExecuteQuery();
  };

  const handlePreviousPage = () => {
    if (currentPage > 1) {
      setCurrentPage((prev) => prev - 1);
      handleExecuteQuery();
    }
  };

  const handleCancelQuery = async () => {
    if (streamIdRef.current) {
      await websocketService.cancelQuery(streamIdRef.current);
      setStatus("Query cancelled");
      setIsStreaming(false);
    }
  };

  const handleLoadColumns = async () => {
    if (!selectedTable) {
      setError("Please select a table first.");
      return;
    }

    setStatus("Loading columns...");
    setError("");

    try {
      const response = await fetch("http://localhost:8080/api/columns", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          table: selectedTable,
          database: parameters.database,
          host: parameters.host,
          port: parseInt(parameters.port, 10),
          username: parameters.username,
          password: parameters.password,
        }),
      });

      if (!response.ok) {
        const errorData = await response.json();
        setError(`Failed to load columns: ${errorData.error}`);
        setStatus("Failed to load columns");
        return;
      }

      const data = await response.json();
      setColumns(data.columns);
      setStatus("Columns loaded");
    } catch (err) {
      setError(`Error loading columns: ${err.message}`);
      setStatus("Failed to load columns");
    }
  };

  const handlePreview = () => {
    setStatus("Previewing data...");
    setTimeout(() => {
      setRows([
        ["value1", "value2", "value3"],
        ["value4", "value5", "value6"],
      ]);
      setStatus("Preview complete");
    }, 1000);
  };

  const handleStartIngestion = () => {
    setStatus("Starting ingestion...");
    setTimeout(() => {
      setResult("Ingestion completed successfully. 100 records ingested.");
      setStatus("Ingestion complete");
    }, 2000);
  };

  return (
    <div className="data-query-container">
      <h2>Data Query and Ingestion</h2>
      <SourceTargetSelector
        sourceType={sourceType}
        setSourceType={setSourceType}
        targetType={targetType}
        setTargetType={setTargetType}
      />
      <ConnectionParameters
        parameters={parameters}
        setParameters={setParameters}
      />
      <TableList
        tables={tables}
        selectedTable={selectedTable}
        setSelectedTable={setSelectedTable}
      />
      <ColumnSelector
        columns={columns}
        selectedColumns={selectedColumns}
        setSelectedColumns={setSelectedColumns}
      />
      <div className="pagination-controls">
        <button onClick={handlePreviousPage} disabled={currentPage === 1}>
          Previous
        </button>
        <span>Page {currentPage}</span>
        <button onClick={handleNextPage}>Next</button>
      </div>
      <ActionButtons
        onConnect={handleConnect}
        onLoadColumns={handleLoadColumns}
        onPreview={handlePreview}
        onStartIngestion={handleStartIngestion}
      />
      <StatusDisplay status={status} error={error} />
      <ResultDisplay result={result} />
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
