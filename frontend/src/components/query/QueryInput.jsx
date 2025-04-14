import React from "react";

const QueryInput = ({ query, setQuery, isStreaming }) => {
  return (
    <div className="query-input">
      <textarea
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        rows={5}
        placeholder="Enter your SQL query here..."
        disabled={isStreaming}
      />
    </div>
  );
};

export default QueryInput;
