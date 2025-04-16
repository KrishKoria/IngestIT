class WebSocketService {
  constructor() {
    this.socket = null;
    this.isConnected = false;
    this.connectionPromise = null;
    this.messageHandlers = new Map();
    this.streamHandlers = new Map();
    this.reconnectAttempts = 0;
    this.maxReconnectAttempts = 5;
    this.reconnectInterval = 3000; // 3 seconds
  }

  connect(url = `ws://${window.location.hostname}:8080/ws`) {
    if (this.connectionPromise) {
      return this.connectionPromise;
    }

    this.connectionPromise = new Promise((resolve, reject) => {
      try {
        console.log(`Connecting to WebSocket at ${url}`);
        this.socket = new WebSocket(url);

        // Set a connection timeout
        const connectionTimeout = setTimeout(() => {
          if (!this.isConnected) {
            console.error("WebSocket connection timeout");
            this.socket.close();
            reject(new Error("Connection timeout"));
          }
        }, 5000); // 5 second timeout

        this.socket.onopen = () => {
          console.log("WebSocket connection established");
          this.isConnected = true;
          this.reconnectAttempts = 0;
          clearTimeout(connectionTimeout);
          resolve(true);
        };

        this.socket.onmessage = (event) => {
          console.log("WebSocket message received:", event.data);
          this.handleMessage(event.data);
        };

        this.socket.onclose = (event) => {
          console.log(
            `WebSocket connection closed: ${event.code} ${event.reason}`
          );
          this.isConnected = false;
          this.connectionPromise = null;
          clearTimeout(connectionTimeout);

          // Attempt to reconnect if not a clean close
          if (
            !event.wasClean &&
            this.reconnectAttempts < this.maxReconnectAttempts
          ) {
            this.reconnectAttempts++;
            console.log(
              `Attempting to reconnect (${this.reconnectAttempts}/${this.maxReconnectAttempts})...`
            );
            setTimeout(() => this.connect(url), this.reconnectInterval);
          }
        };

        this.socket.onerror = (error) => {
          console.error("WebSocket error:", error);
          clearTimeout(connectionTimeout);
          reject(error);
        };
      } catch (error) {
        console.error("Error creating WebSocket:", error);
        this.connectionPromise = null;
        reject(error);
      }
    });

    return this.connectionPromise;
  }

  // Ensure connection is established
  ensureConnected() {
    if (this.isConnected) {
      return Promise.resolve(true);
    }
    return this.connect();
  }

  async send(message) {
    await this.ensureConnected();

    if (typeof message === "object") {
      message = JSON.stringify(message);
    }

    console.log("Sending message to WebSocket:", message);
    this.socket.send(message);
  }

  async executeQuery(query, streamId = crypto.randomUUID()) {
    console.log("Executing query via WebSocket:", query);
    await this.send({
      type: "query",
      query: query,
      streamId: streamId,
    });

    return streamId;
  }

  // Cancel a query
  async cancelQuery(streamId) {
    await this.send({
      type: "cancelQuery",
      streamId: streamId,
    });
  }

  // Register a handler for stream events
  onStream(streamId, handlers) {
    this.streamHandlers.set(streamId, handlers);
    return () => this.streamHandlers.delete(streamId);
  }

  // Register a handler for a specific message type
  onMessageType(type, handler) {
    if (!this.messageHandlers.has(type)) {
      this.messageHandlers.set(type, []);
    }

    this.messageHandlers.get(type).push(handler);

    // Return unsubscribe function
    return () => {
      const handlers = this.messageHandlers.get(type);
      if (handlers) {
        const index = handlers.indexOf(handler);
        if (index !== -1) {
          handlers.splice(index, 1);
        }
      }
    };
  }

  // Handle incoming messages
  handleMessage(data) {
    try {
      const message = JSON.parse(data);

      // Handle based on message type
      if (message.type) {
        // Dispatch to type-specific handlers
        const typeHandlers = this.messageHandlers.get(message.type);
        if (typeHandlers) {
          typeHandlers.forEach((handler) => handler(message));
        }

        // Dispatch to stream-specific handlers if streamId is present
        if (message.streamId && this.streamHandlers.has(message.streamId)) {
          const streamHandler = this.streamHandlers.get(message.streamId);
          if (streamHandler[message.type]) {
            streamHandler[message.type](message);
          }

          // Handle 'complete' or 'error' message by cleaning up
          if (message.type === "complete" || message.type === "error") {
            this.streamHandlers.delete(message.streamId);
          }
        }
      }
    } catch (error) {
      console.error("Error parsing WebSocket message:", error, data);
    }
  }

  // Close the connection
  disconnect() {
    if (this.socket && this.isConnected) {
      this.socket.close(1000, "Client disconnecting");
      this.isConnected = false;
      this.connectionPromise = null;
    }
  }
}

// Create singleton instance
const websocketService = new WebSocketService();
export default websocketService;
