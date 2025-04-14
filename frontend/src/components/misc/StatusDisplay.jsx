import React from "react";

const StatusDisplay = ({ status, error }) => {
  return (
    <div className="status-display">
      {status && <div className="status">{status}</div>}
      {error && <div className="error">{error}</div>}
    </div>
  );
};

export default StatusDisplay;
