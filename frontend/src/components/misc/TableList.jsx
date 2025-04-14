import React from "react";

const TableList = ({ tables, selectedTable, setSelectedTable }) => {
  return (
    <div className="table-list">
      <h3>Available Tables</h3>
      <ul>
        {tables.map((table) => (
          <li key={table}>
            <label>
              <input
                type="radio"
                name="table"
                value={table}
                checked={selectedTable === table}
                onChange={() => setSelectedTable(table)}
              />
              {table}
            </label>
          </li>
        ))}
      </ul>
    </div>
  );
};

export default TableList;
