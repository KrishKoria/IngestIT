# IngestIT

IngestIT is a data ingestion and querying platform designed to interact with ClickHouse databases.
It provides a user-friendly interface for querying data, selecting specific columns, and managing data ingestion workflows.
The platform uses WebSocket communication for real-time query execution and streaming results.

## Features

- **WebSocket-Based Query Execution**: Real-time query execution and result streaming using WebSocket.
- **Dynamic Column Selection**: Select specific columns from a table to fetch only the required data.
- **Pagination Support**: Fetch data in chunks using pagination to improve performance.
- **ClickHouse Integration**: Seamless integration with ClickHouse for querying and data ingestion.
- **File Export**: Export query results to files with progress tracking (future feature).
- **Error Handling**: Robust error handling for WebSocket communication and query execution.

## Project Structure

`plaintext
 c:\Users\Krish Koria\Desktop\projects\IngestIT\
 ├── backend\
 │   ├── main.go                 Entry point for the backend server.
 │   ├── websocket.go            WebSocket handler for real-time query execution.
 │   ├── db.go                   ClickHouse database connection logic.
 │   ├── file_ingestion.go       File export and ingestion logic (future feature).
 ├── frontend\
 │   ├── src\
 │   │   ├── components\
 │   │   │   ├── DataQuery.jsx   Main component for data querying and ingestion.
 │   │   │   ├── selectors\
 │   │   │   │   ├── ColumnSelector.jsx  Component for selecting columns.
 │   │   │   ├── query\
 │   │   │   │   ├── QueryControls.jsx   Controls for executing and canceling queries.
 │   │   ├── services\
 │   │   │   ├── websocketservice.js     WebSocket service for frontend-backend communication.
 ├── Readme.md                  Project documentation.
 `

## Prerequisites

- **Backend**:
- Go 1.18+
- ClickHouse database
- Gin framework
- Gorilla WebSocket library
- **Frontend**:
- Node.js 16+
- React.js

## Installation

### Backend

1.  Navigate to the `backend` directory:
    `bash
   cd backend
   `
2.  Install dependencies:
    `bash
   go mod tidy
   `
3.  Run the backend server:
    `bash
   go run main.go
   `

### Frontend

1.  Navigate to the `frontend` directory:
    `bash
   cd frontend
   `
2.  Install dependencies:
    `bash
   npm install
   `
3.  Start the frontend development server:
    `bash
   npm start
   `

## Usage

1.  Open the frontend application in your browser (default: `http:localhost:3000`).
2.  Connect to the ClickHouse database by providing the connection parameters.
3.  Select a table and load its columns.
4.  Choose specific columns to query and click "Execute Query".
5.  View the results in the table or export them to a file (future feature).

## API Endpoints

### Backend

- **WebSocket Endpoint**:
- `GET /ws`: WebSocket endpoint for real-time query execution.
- **REST Endpoints**:
- `POST /api/connect`: Connect to the ClickHouse database.
- `POST /api/query`: Execute a query with pagination support.
- `POST /api/columns`: Fetch columns for a specific table.

## WebSocket Message Structure

- **Query Message**:
  `json
  {
    "type": "query",
    "query": "SELECT * FROM table_name",
    "streamId": "unique-stream-id"
  }
  `
- **Cancel Query Message**:
  `json
  {
    "type": "cancelQuery",
    "streamId": "unique-stream-id"
  }
  `

## Troubleshooting

- **Frontend Stuck at "Executing Query"**:
- Ensure the WebSocket connection is established.
- Check the backend logs for received messages.
- **Backend Not Receiving Messages**:
- Verify the WebSocket route (`/ws`) is registered in `main.go`.
- Add debugging logs in `readPump` to confirm message receipt.

## Future Enhancements

- Add support for exporting query results to files.
- Implement advanced filtering and sorting options.
- Enhance error reporting and logging.
- Add authentication and authorization for secure access.

## Contributing

Contributions are welcome! Please fork the repository and submit a pull request.

## License

This project is licensed under the MIT License.
