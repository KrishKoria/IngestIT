import React from "react";

const ConnectionParameters = ({ parameters, setParameters }) => {
  const handleChange = (key, value) => {
    setParameters((prev) => ({ ...prev, [key]: value }));
  };

  return (
    <div className="connection-parameters">
      <h3>Connection Parameters</h3>
      <div>
        <label>Host:</label>
        <input
          type="text"
          value={parameters.host || ""}
          onChange={(e) => handleChange("host", e.target.value)}
        />
      </div>
      <div>
        <label>Port:</label>
        <input
          type="number"
          value={parameters.port || ""}
          onChange={(e) => handleChange("port", e.target.value)}
        />
      </div>
      <div>
        <label>Database:</label>
        <input
          type="text"
          value={parameters.database || ""}
          onChange={(e) => handleChange("database", e.target.value)}
        />
      </div>
      <div>
        <label>Username:</label>
        <input
          type="text"
          value={parameters.username || ""}
          onChange={(e) => handleChange("username", e.target.value)}
        />
      </div>
      <div>
        <label>Password:</label>
        <input
          type="password"
          value={parameters.password || ""}
          onChange={(e) => handleChange("password", e.target.value)}
        />
      </div>
    </div>
  );
};

export default ConnectionParameters;
