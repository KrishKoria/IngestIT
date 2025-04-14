import React from "react";

const ActionButtons = ({
  onConnect,
  onLoadColumns,
  onPreview,
  onStartIngestion,
}) => {
  return (
    <div className="action-buttons">
      <button onClick={onConnect}>Connect</button>
      <button onClick={onLoadColumns}>Load Columns</button>
      <button onClick={onPreview}>Preview</button>
      <button onClick={onStartIngestion}>Start Ingestion</button>
    </div>
  );
};

export default ActionButtons;
