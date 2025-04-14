import React from "react";

const ColumnSelector = ({ columns, selectedColumns, setSelectedColumns }) => {
  const toggleColumn = (column) => {
    setSelectedColumns((prev) =>
      prev.includes(column)
        ? prev.filter((col) => col !== column)
        : [...prev, column]
    );
  };

  return (
    <div className="column-selector">
      <h3>Columns</h3>
      <ul>
        {columns.map((column) => (
          <li key={column}>
            <label>
              <input
                type="checkbox"
                value={column}
                checked={selectedColumns.includes(column)}
                onChange={() => toggleColumn(column)}
              />
              {column}
            </label>
          </li>
        ))}
      </ul>
    </div>
  );
};

export default ColumnSelector;
