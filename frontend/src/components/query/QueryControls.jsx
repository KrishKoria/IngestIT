import React from "react";

const QueryControls = ({ isStreaming, onExecute, onCancel }) => {
  return (
    <div className="query-controls">
      {!isStreaming ? (
        <button onClick={onExecute}>Execute Query</button>
      ) : (
        <button onClick={onCancel}>Cancel Query</button>
      )}
    </div>
  );
};

export default QueryControls;
