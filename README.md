# 🕰️ Cron Discover Service

This Go-based service manages cron jobs within workspaces, providing REST APIs for cron and webhook management. It uses the official crontab format and notifies registered webhooks when crons are triggered.

## 🌟 Features

- Cron management (create, list, delete)
- Webhook management (add, list, remove)
- Workspace-based SQLite database for cron execution logs
- Automatic notification of registered webhooks when crons are triggered

## 🛠 Prerequisites

- Go 1.16 or higher
- SQLite3

## 🚀 Installation

1. Clone the repository:

   ```
   git clone https://github.com/yourusername/cron-discover.git
   cd cron-discover
   ```

2. Install dependencies:
   ```
   go get github.com/gorilla/mux
   go get github.com/mattn/go-sqlite3
   go get github.com/robfig/cron/v3
   ```

## 🏃‍♂️ Usage

1. Build the service:

   ```
   make build
   ```

2. Run the service:
   ```
   make run
   ```

The service will start on port 8080 by default. You can specify a different port using the `-port` flag:

```
./cron-discover -port 9000
```

## 🔧 API Endpoints

### Crons

- Create a new cron:

  ```
  curl -X POST http://localhost:8080/crons \
    -H "Content-Type: application/json" \
    -d '{"workspace_id": 1, "name": "My Cron", "description": "Description of my cron", "cron_expression": "*/5 * * * *"}'
  ```

- List all crons in a workspace:

  ```
  curl "http://localhost:8080/crons?workspace_id=1"
  ```

- Delete a cron:
  ```
  curl -X DELETE http://localhost:8080/crons/1
  ```

### Webhooks

- Add a webhook to a cron:

  ```
  curl -X POST http://localhost:8080/crons/1/webhooks \
    -H "Content-Type: application/json" \
    -d '{"url": "https://example.com/webhook"}'
  ```

- List webhooks for a cron:

  ```
  curl http://localhost:8080/crons/1/webhooks
  ```

- Remove a webhook:
  ```
  curl -X DELETE http://localhost:8080/crons/1/webhooks/1
  ```

## 📊 Cron Execution Logs

Cron execution logs are stored in workspace-specific SQLite databases located at `/home/workspaces/{workspace_id}/cron.sqlite`. Each execution is logged with a status of "started", "completed", or "failed".

## 🧑‍💻 Development

To run the service in development mode with automatic reloading:

```
make dev
```

## 🧪 Testing

Run the test suite:

```
make test
```

## 🧹 Cleaning up

Remove the built binary:

```
make clean
```

## 📚 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
