import React from "react";

const QueryResults = ({ columns, rows, isStreaming, isLoadingData }) => {
  return (
    <div className="query-results">
      <h3>Results {isStreaming && <span>(Streaming...)</span>}</h3>
      <div className="results-table-container">
        {isLoadingData && rows.length === 0 && (
          <div className="loading-indicator">
            Waiting for data... (This might take a moment for large datasets)
          </div>
        )}
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
  );
};

export default QueryResults;
