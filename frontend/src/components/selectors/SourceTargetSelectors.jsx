import React from "react";

const SourceTargetSelector = ({
  sourceType,
  setSourceType,
  targetType,
  setTargetType,
}) => {
  return (
    <div className="source-target-selector">
      <div>
        <label>Source Type:</label>
        <select
          value={sourceType}
          onChange={(e) => setSourceType(e.target.value)}
        >
          <option value="clickhouse">ClickHouse</option>
          <option value="flatfile">Flat File</option>
        </select>
      </div>
      <div>
        <label>Target Type:</label>
        <select
          value={targetType}
          onChange={(e) => setTargetType(e.target.value)}
        >
          <option value="clickhouse">ClickHouse</option>
          <option value="flatfile">Flat File</option>
        </select>
      </div>
    </div>
  );
};

export default SourceTargetSelector;
