import React from "react";

const ResultDisplay = ({ result }) => {
  return (
    <div className="result-display">
      <h3>Result</h3>
      <div>{result}</div>
    </div>
  );
};

export default ResultDisplay;
