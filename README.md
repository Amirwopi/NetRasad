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
* **🖥️ Taskbar & System Tray Overlay**: Keep an eye on your network throughput without maximizing the app. Includes support for multi-monitor taskbar displays on Windows; on Linux the live speed sits next to the tray icon.
* **🔍 Per-Application Monitoring**: View exact data usage and active network connections grouped by individual processes.
* **🛠️ Advanced Network Diagnostics**:
  * **Ping**, **Traceroute**, **DNS Lookup**, **TCP Connect**
  * Gateway and Interface detection (with SSID integration).
* **🌍 Full i18n Support**: Native RTL (Persian) and LTR (English) localization.
* **💾 Persistent Storage & Quotas**: All historical data is safely stored locally via SQLite, providing accurate Data Quota tracking across reboots.
* **📡 Modem/Router Integration**: Direct Telnet integration to view ADSL stats, WiFi configurations (SSID and ghosts), active DNS, Firewall/NAT rules, and Reboot capability.
* **🎨 Modern UI/UX**: Fully customizable dark-themed UI, flexible color schemes, draggable Ping widget, and seamless native-like experience.

---

## 📸 Screenshots

Here is a glimpse of NetRasad in action:

<details>
<summary><b>Click to Expand Screenshots Gallery</b></summary>

| Dashboard | Applications Monitor |
| :---: | :---: |
| <img src="screenshot/Dashboard.png" width="400"> | <img src="screenshot/Applications.png" width="400"> |

| Connections | Network Diagnostics |
| :---: | :---: |
| <img src="screenshot/connections.png" width="400"> | <img src="screenshot/diagnostics.png" width="400"> |

| Settings & Customization | Tooltip & Widget Preview |
| :---: | :---: |
| <img src="screenshot/Settings.png" width="400"> | <img src="screenshot/Tooltip_Widget.png" width="400"> |

*Additional previews:*
<br>
**Traffic Reports:**<br>
<img src="screenshot/Traffic_Reports.png" width="400">
<br>
**Data Quota & Others:**<br>
<img src="screenshot/Data_Quota.png" width="400">
<img src="screenshot/Widget_Ping.png" width="200">

</details>

---

## ⚙️ Tech Stack

| Layer | Technology |
| :--- | :--- |
| **Backend Core** | Go 1.26, Wails v2 |
| **Frontend UI** | Vue 3, TypeScript, Vite, Pinia, Vue-Router, Vue-i18n |
| **Data Storage** | SQLite |
| **Platform APIs** | Windows: IP Helper (`GetIfTable2`), `wlanapi.dll` · Linux: sysfs/procfs, rtnetlink & nl80211, D-Bus (StatusNotifierItem tray) |

---

## 🗺️ Roadmap & TODOs

- [x] Multi-monitor taskbar overlay support.
- [x] Localization (English/Persian).
- [x] **Modem Settings Integration** - Add configuration pages and features specifically for managing and viewing Modem/Router statistics directly from NetRasad.
- [x] Floating Ping Widget with persistent dragging support.
- [x] Linux support (tray icon with live speed, autostart, `.deb`/`.rpm` packages).

---

## 🛠️ Build & Installation

### Prerequisites
* **Go** 1.21+ (1.26 recommended)
* **Node.js** 18+
* **Wails CLI** v2
* **Linux only:** GTK 3 and WebKitGTK 4.1 development files

| Distribution | Command |
| :--- | :--- |
| Ubuntu 22.04+ / Debian 12+ | `sudo apt install build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.1-dev` |
| Fedora | `sudo dnf install gcc pkgconf-pkg-config gtk3-devel webkit2gtk4.1-devel` |
| Arch Linux | `sudo pacman -S --needed base-devel gtk3 webkit2gtk-4.1` |

### Setup
Install Wails CLI:
```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### Development Mode
To run the app with live reload for both backend and frontend:
```bash
wails dev                    # Windows
wails dev -tags webkit2_41   # Linux
```

### Production Build
To compile the final standalone executable:
```bash
wails build -clean -platform windows/amd64
```

Alternatively, use the custom PowerShell build script to inject version tags (e.g. for GitHub releases):
```powershell
.\scripts\build.ps1 -Version v1.0.2
```
The output binary will be located at: `build/bin/NetRasad.exe`

### Linux Build
```bash
wails build -clean -platform linux/amd64 -tags webkit2_41
```
Or use the build script, which injects the version and can also produce the release packages (`--package` needs [nfpm](https://nfpm.goreleaser.com)):
```bash
./scripts/build.sh v1.0.2             # build/bin/NetRasad
./scripts/build.sh v1.0.2 --package   # build/dist/: .tar.gz, .deb and .rpm
```

### Installing on Linux
Every [release](https://github.com/Amirwopi/NetRasad/releases) ships a `.deb`, an `.rpm` and a portable `.tar.gz`. The packages pull in GTK and WebKitGTK automatically:
```bash
sudo apt install ./netrasad_*_amd64.deb      # Ubuntu 22.04+, Debian 12+
sudo dnf install ./netrasad-*.x86_64.rpm     # Fedora
```
From the tarball, install `libwebkit2gtk-4.1` (and GTK 3) with your package manager, then run `./NetRasad`.

**Linux notes**
* **Tray icon:** shown on KDE Plasma, Cinnamon, XFCE, MATE, Budgie and panels such as waybar. Stock GNOME needs the [AppIndicator extension](https://extensions.gnome.org/extension/615/appindicator-support/) (preinstalled on Ubuntu). Without a tray, closing the window quits the app instead of hiding it.
* **Live speed in the panel:** the label next to the tray icon is displayed by hosts that support indicator labels (GNOME with AppIndicator, Ubuntu); elsewhere it appears in the icon's tooltip. The Windows-only floating overlays (taskbar band, ping widget) are not available on Linux.
* **Diagnostics:** ping uses `ping`; traceroute uses `traceroute` when installed and falls back to `tracepath` (iputils).
* **Per-process ownership:** sockets owned by other users (e.g. system daemons) are listed without a process name unless NetRasad runs as root.

---

## 📄 License
This project is licensed under the [MIT License](LICENSE).
