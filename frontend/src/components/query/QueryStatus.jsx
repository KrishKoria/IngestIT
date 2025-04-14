import React from "react";

const QueryStatus = ({ status, error }) => {
  return (
    <div className="query-status-container">
      {status && <div className="query-status">{status}</div>}
      {error && <div className="query-error">{error}</div>}
    </div>
  );
};

export default QueryStatus;
