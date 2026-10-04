# 🌐 NetRasad (نت‌رصد)

<div align="center">
  <img src="https://img.shields.io/badge/Go-1.26-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go" />
  <img src="https://img.shields.io/badge/Vue.js-3-42b883?style=for-the-badge&logo=vue.js&logoColor=white" alt="Vue 3" />
  <img src="https://img.shields.io/badge/Wails-v2-ff0000?style=for-the-badge&logo=wails&logoColor=white" alt="Wails" />
  <img src="https://img.shields.io/badge/SQLite-storage-003b57?style=for-the-badge&logo=sqlite&logoColor=white" alt="SQLite" />
  <img src="https://img.shields.io/badge/Platform-Windows%20%7C%20Linux-blue?style=for-the-badge" alt="Platform" />
  <img src="https://img.shields.io/badge/License-MIT-green?style=for-the-badge" alt="License" />
</div>

<br />

> **NetRasad** is an advanced, cross-platform desktop application designed for real-time network traffic monitoring and deep packet diagnostics. Built on top of the highly performant **Wails v2** framework (Go backend + Vue 3 frontend), it seamlessly provides live speed graphs, advanced diagnostic tools, per-app connection viewers, and an intuitive UI. 

---

## 🚀 Key Features

* **⚡ Real-Time Traffic Monitoring**: Track your network download and upload speeds with a beautiful, dynamic live graph.
* **🖥️ Taskbar & System Tray Overlay**: Keep an eye on your network throughput without maximizing the app. Includes support for multi-monitor taskbar displays.
* **🔍 Per-Application Monitoring**: View exact data usage and active network connections grouped by individual processes.
* **🛠️ Advanced Network Diagnostics**:
  * **Ping**, **Traceroute**, **DNS Lookup**, **TCP Connect**
  * Gateway and Interface detection (with SSID integration).
* **🌍 Full i18n Support**: Native RTL (Persian) and LTR (English) localization.
* **💾 Persistent Storage**: All historical data is safely stored locally via SQLite for future analysis and reporting.
* **🎨 Modern UI/UX**: Fully customizable dark-themed UI, flexible color schemes, and seamless native-like experience.

---

## 📸 Screenshots

Here is a glimpse of NetRasad in action:

<details>
<summary><b>Click to Expand Screenshots Gallery</b></summary>

| Dashboard | Applications Monitor |
| :---: | :---: |
| <img src="screenshot/NetRasad_5W6IeUGX3q.png" width="400"> | <img src="screenshot/NetRasad_duDBrIJAX2.png" width="400"> |

| Connections | Network Diagnostics |
| :---: | :---: |
| <img src="screenshot/NetRasad_6EFyM6y3dg.png" width="400"> | <img src="screenshot/NetRasad_wtvf5aGNho.png" width="400"> |

| Settings & Customization | Tooltip & Widget Preview |
| :---: | :---: |
| <img src="screenshot/NetRasad_23YVk06Xon.png" width="400"> | <img src="screenshot/NetRasad_p4pnCUbDx0.png" width="400"> |

*Additional previews:*
<br>
**Traffic Reports:**<br>
<img src="screenshot/NetRasad_SFSX8bQccw.png" width="400">
<br>
**Data Quota:**<br>
<img src="screenshot/NetRasad_pdedxazoRu.png" width="400">

</details>

---

## ⚙️ Tech Stack

| Layer | Technology |
| :--- | :--- |
| **Backend Core** | Go 1.26, Wails v2 |
| **Frontend UI** | Vue 3, TypeScript, Vite, Pinia, Vue-Router, Vue-i18n |
| **Data Storage** | SQLite |
| **Platform APIs** | Windows IP Helper (`GetIfTable2`), `wlanapi.dll` |

---

## 🗺️ Roadmap & TODOs

- [x] Multi-monitor taskbar overlay support.
- [x] Localization (English/Persian).
- [ ] **TODO: Modem Settings Integration** - Add configuration pages and features specifically for managing and viewing Modem/Router statistics directly from NetRasad.

---

## 🛠️ Build & Installation

### Prerequisites
* **Go** 1.21+ (1.26 recommended)
* **Node.js** 18+
* **Wails CLI** v2

### Setup
Install Wails CLI:
```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### Development Mode
To run the app with live reload for both backend and frontend:
```bash
wails dev
```

### Production Build
To compile the final standalone executable:
```bash
wails build -clean -platform windows/amd64
```
The output binary will be located at: `build/bin/NetRasad.exe`

---

## 📄 License
This project is licensed under the [MIT License](LICENSE).
