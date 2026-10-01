# NetRasad

![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)
![Vue](https://img.shields.io/badge/Vue-3-42b883?logo=vue.js&logoColor=white)
![Wails](https://img.shields.io/badge/Wails-v2-ff0000?logo=wails&logoColor=white)
![SQLite](https://img.shields.io/badge/SQLite-storage-003b57?logo=sqlite&logoColor=white)
![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20Linux-blue)
![License](https://img.shields.io/badge/License-MIT-green)

> **NetRasad** is a cross-platform desktop application for real-time network
> traffic monitoring. Built with Wails v2 (Go backend + Vue 3 frontend), it
> provides live speed graphing, network diagnostics tools, an active
> connections viewer, and persistent SQLite storage — all in a dark-themed,
> internationalized UI.

---

## Features

- **Real-time traffic monitoring** — download/upload speed and total bytes
  transferred, updated live with a smooth speed graph.
- **Active interface detection** — only connected network interfaces are
  listed; disconnected adapters are hidden automatically.
- **Network selection in Settings** — choose which networks to monitor from
  the detected active interfaces.
- **WiFi SSID + gateway IP display** — shows the connected SSID and gateway
  IP address per interface.
- **System tray icon** — a tray icon with a live speed overlay so you can
  glance at throughput without opening the window.
- **Taskbar overlay icon** — Windows taskbar overlay icon displaying
  download/upload speed.
- **Diagnostics tools**:
  - Ping
  - Traceroute
  - DNS Lookup
  - TCP Connect
  - Gateway detection
- **Active connections viewer** — lists all active TCP/UDP connections with
  process attribution (PID + process name).
- **SQLite storage** — traffic history is persisted locally in a SQLite
  database for historical analysis.
- **i18n** — full internationalization with English (LTR) and Persian (RTL)
  translations.
- **Dark theme UI** — a polished dark theme throughout the application.
- **CMD window hiding on Windows** — child processes (ping, tracert, etc.)
  are spawned with `CREATE_NO_WINDOW` so no console windows flash on screen.

---

## Tech Stack

| Layer        | Technology |
|--------------|------------|
| **Backend**  | Go 1.26, Wails v2 |
| **Frontend** | Vue 3, TypeScript, Vite, Pinia, vue-i18n, vue-router |
| **Storage**  | SQLite |
| **Platform APIs** | Windows IP Helper (`GetIfTable2`, `GetAdaptersAddresses`), `wlanapi.dll` |

---

## Project Structure

```
NetRasad/
├── build/                  # Build output and platform assets
│   └── bin/                # Compiled binaries
├── frontend/               # Vue 3 frontend application
│   ├── src/
│   │   ├── components/      # Reusable UI components
│   │   ├── views/           # Page-level views
│   │   ├── stores/          # Pinia stores
│   │   ├── i18n/            # English + Persian translations
│   │   ├── router/          # vue-router configuration
│   │   └── assets/          # Static assets (icons, styles)
│   ├── package.json
│   └── vite.config.ts
├── app.go                  # Wails application entry
├── main.go                 # Go main entry point
├── wails.json              # Wails project configuration
├── go.mod
└── README.md
```

---

## Build

### Prerequisites

- **Go** 1.21+ (1.26 recommended)
- **Node.js** 18+
- **Wails CLI** v2

### Install Wails CLI

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### Build

```bash
wails build -clean -platform windows/amd64
```

Output binary:

```
build/bin/NetRasad.exe
```

### Development Mode

```bash
wails dev
```

This starts the Wails dev server with hot-reload for the frontend and
rebuilds the Go backend on change.

---

## Screenshots

<!-- TODO: add screenshots -->

---

## License

MIT
