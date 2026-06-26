# FFXIV Data Collector

A Go-based application that connects to an Advanced Combat Tracker (ACT) WebSocket to collect, parse, and store Final Fantasy XIV encounter and travel log data in various databases.

## Features
- Real-time event tracking over WebSocket from ACT OverlayPlugin.
- Tracks Zone Changes (Travel Log).
- Tracks Combat Encounters (DPS, Healing, Damage Taken, Deaths per player).
- Supports multiple local and remote SQL database engines.

## Installation

Ensure you have Go installed on your system.

```bash
# Clone the repository and build the project
git clone <repository_url>
cd ffxiv_data_collector
go build .
```

## Usage

Run the executable to start the collector. 

```bash
./ffxiv_data_collector
```

### Command-line Flags

The application supports the following command-line flags:

- `--debug`: Enables raw event printing to the console for troubleshooting and development.
  - Example: `./ffxiv_data_collector --debug`
- `--portable`: Forces the application to create and load `config.json` from the current working directory instead of the OS default configuration path.
  - Example: `./ffxiv_data_collector --portable`

## Configuration

On the first launch, the application will automatically generate a default `config.json` file. Depending on your operating system (and if the `--portable` flag is omitted), it will be placed in the following location:

- **Windows**: `%LOCALAPPDATA%\ffxiv_data_collector\config.json`
- **Linux / macOS**: `~/.config/ffxiv_data_collector/config.json`

### Default Config
```json
{
  "database": {
    "type": "sqlite",
    "dsn": "ffxiv_events.db"
  },
  "websocket_url": "ws://127.0.0.1:10501/ws"
}
```

### Environment Variables

You can override the configuration file values by setting the following environment variables. This is particularly useful when running the application in a Docker container.

- `FFXIV_DB_TYPE`: Overrides `database.type`
- `FFXIV_DB_DSN`: Overrides `database.dsn`
- `FFXIV_WS_URL`: Overrides `websocket_url`

## Database Connection Settings

The collector relies on standard SQL drivers. You can configure it to connect to different engines by modifying the `"type"` and `"dsn"` (Data Source Name) fields in your `config.json`.

Supported `"type"` values: `sqlite`, `postgres` (or `postgresql`), `mysql`, `sqlserver` (or `mssql`), `libsql` (or `turso`).

### SQLite (Default)
`"type": "sqlite"`
- **DSN Format:** `ffxiv_events.db` (or an absolute path like `C:/path/to/db.sqlite`)

### PostgreSQL
`"type": "postgres"`
- **DSN Format (URL):** `postgres://user:password@localhost:5432/dbname?sslmode=disable`
- **DSN Format (Key/Value):** `user=postgres password=mysecretpassword dbname=mydb sslmode=disable`

### MySQL
`"type": "mysql"`
- **DSN Format:** `user:password@tcp(127.0.0.1:3306)/dbname`

### Microsoft SQL Server
`"type": "sqlserver"`
- **DSN Format:** `sqlserver://username:password@localhost:1433?database=dbname`

### libSQL / Turso Network
`"type": "libsql"`
- **DSN Format (Remote):** `libsql://your-db-name-your-org.turso.io?authToken=your-auth-token`
- **DSN Format (Local):** `http://127.0.0.1:8080`
