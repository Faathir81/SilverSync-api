# 🐺 SilverSync API (Backend)

![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)
![Python Version](https://img.shields.io/badge/Python-3.8+-3776AB?style=flat&logo=python)
![License](https://img.shields.io/badge/License-MIT-green.svg)

SilverSync API is the core backend engine designed to bridge Spotify metadata, local CLI downloading via `spotDL`, and cloud storage distribution via Google Drive API. It acts as the processing server to download high-quality audio and serve it to a mobile client for offline playback.

## 🏗️ System Architecture

1. **Request:** Client sends a Spotify Playlist/Track URL to this Go API.
2. **Process:** API executes `spotDL` (Python CLI) to download the track and attach metadata & album art.
3. **Upload:** API uploads the processed `.mp3` file to a dedicated Google Drive folder via Service Account.
4. **Record:** API stores the Google Drive File ID and track metadata into the database.
5. **Clean:** API deletes the temporary local file to save server storage.
6. **Serve:** API serves a JSON list of available tracks and their direct download links for the Flutter app.

## 🛠️ Tech Stack

* **Language:** Golang (Go)
* **Downloader Engine:** Python + spotDL + FFmpeg
* **Cloud Storage:** Google Drive API v3 (Service Account)
* **Database:** MySQL / PostgreSQL

---

## 🗺️ Development Roadmap

This project is built in iterative phases to ensure stability and performance:

### Phase 1: Foundation & Setup ⏳ (In Progress)
- [ ] Initialize Go module and project directory structure (Clean Architecture).
- [ ] Set up environment variables (`.env`) for database credentials and API keys.
- [ ] Design and implement database schema (ERD) for `tracks` and `sync_logs`.
- [ ] Create basic REST API router using `gin-gonic/gin` or standard `net/http`.

### Phase 2: Core Engine Integration (spotDL) 📝 (Planned)
- [ ] Implement Go wrapper to execute shell commands (`os/exec`).
- [ ] Create function to trigger `spotDL` with specific arguments (download path, format).
- [ ] Handle CLI stdout/stderr to parse download progress and errors.
- [ ] Implement Goroutines for background processing so the API doesn't timeout.

### Phase 3: Cloud Integration (Google Drive) 📝 (Planned)
- [ ] Set up Google Cloud Console Project and generate Service Account JSON.
- [ ] Integrate `google.golang.org/api/drive/v3` into the Go project.
- [ ] Create an upload function that takes the local downloaded `.mp3` and pushes it to Drive.
- [ ] Implement `defer os.Remove()` for automatic temporary file cleanup after upload.

### Phase 4: API Endpoints Construction 📝 (Planned)
- [ ] `POST /api/v1/sync`: Accept Spotify URL, initiate background download & upload worker.
- [ ] `GET /api/v1/tracks`: Retrieve all synced tracks and their Google Drive File IDs from the database.
- [ ] `GET /api/v1/status`: Check the status of ongoing background download tasks.

### Phase 5: Optimization & Refactoring 📝 (Planned)
- [ ] Implement a **Worker Pool** system to limit concurrent `spotDL` executions and prevent CPU/RAM overload.
- [ ] Add robust error handling and retry mechanisms for Google Drive API rate limits.
- [ ] Implement logging (e.g., `logrus` or `zap`) to monitor background task health.

---

## ⚙️ Prerequisites

To run this project locally, you need to install:
1. **Golang** (v1.21 or newer)
2. **Python** (v3.8 or newer)
3. **FFmpeg** (Added to System PATH)
4. **spotDL** (Installed globally via `pip install spotdl`)
5. **Google Service Account Key** (JSON file placed in the root directory)

## 🚀 How to Run (Local Development)
```bash
# Clone the repository
git clone [https://github.com/yourusername/silversync-api.git](https://github.com/yourusername/silversync-api.git)

# Go to project directory
cd silversync-api

# Install dependencies
go mod tidy

# Run the server
go run main.go
