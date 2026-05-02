# 🐺 SilverSync API (Backend)

![Go Version](https://img.shields.io/badge/Go-1.25.0-00ADD8?style=flat&logo=go)
![Python Version](https://img.shields.io/badge/Python-3.10.6-3776AB?style=flat&logo=python)
![License](https://img.shields.io/badge/License-MIT-green.svg)

SilverSync API is the core backend engine designed to bridge Spotify metadata, local CLI downloading via `yt-dlp`, and cloud storage distribution via Google Drive API. It acts as the processing server to download high-quality audio and serve it to a mobile client for offline playback.

## 🏗️ System Architecture

1. **Request:** Client sends a Spotify Playlist/Track URL to this Go API.
2. **Metadata:** API fetches track metadata (Title, Artist, Album Art) securely via the official Spotify Web API.
3. **Process:** API executes `yt-dlp` to download the track audio from YouTube and injects ID3 tags (metadata & cover art) directly in Go.
4. **Upload:** API uploads the processed `.mp3` file to a dedicated Google Drive folder via Service Account.
5. **Record:** API stores the Google Drive File ID and track metadata into the database.
6. **Clean:** API deletes the temporary local file to save server storage.
7. **Serve:** API serves a JSON list of available tracks and their direct download links for the Flutter app.

## 🛠️ Tech Stack

- **Language:** Golang (Go)
- **Downloader Engine:** yt-dlp + FFmpeg
- **Metadata API:** Spotify Web API (zmb3/spotify/v2)
- **Cloud Storage:** Google Drive API v3 (Service Account)
- **Database:** PostgreSQL

---

## 🗺️ Development Roadmap

This project is built in iterative phases to ensure stability and performance:

### Phase 1: Foundation & Setup ⏳

- [x] Initialize Go module and project directory structure (Clean Architecture).
- [x] Set up environment variables (`.env`) for database credentials and API keys.
- [x] Design and implement database schema (ERD) for `tracks` and `sync_logs`.
- [x] Create basic REST API router using `gin-gonic/gin` or standard `net/http`.

### Phase 2: Core Engine Integration (Spotify API & yt-dlp) 📝

- [x] Implement Spotify API client (`zmb3/spotify/v2`) to securely fetch playlist/track metadata.
- [x] Implement Go wrapper to execute `yt-dlp` commands with anti-rate-limit arguments (cookies, sleep intervals).
- [x] Implement ID3 tag injection (`bogem/id3v2`) to embed downloaded metadata into the MP3 file.
- [x] Implement Goroutines for background processing so the API doesn't timeout.

### Phase 3: Cloud Integration (Google Drive) 📝

- [x] Set up Google Cloud Console Project and generate Service Account JSON.
- [x] Integrate `google.golang.org/api/drive/v3` into the Go project.
- [x] Create an upload function that takes the local downloaded `.mp3` and pushes it to Drive.
- [x] Implement `defer os.Remove()` for automatic temporary file cleanup after upload.

### Phase 4: API Endpoints Construction 📝

- [x] `POST /api/v1/sync`: Accept Spotify URL, initiate background download & upload worker.
- [x] `GET /api/v1/tracks`: Retrieve all synced tracks and their Google Drive File IDs from the database.
- [x] `GET /api/v1/status`: Check the status of ongoing background download tasks.

### Phase 5: Optimization & Refactoring 📝

- [ ] Implement a **Worker Pool** system to limit concurrent `yt-dlp` executions and prevent CPU/RAM/Rate-Limit overload.
- [ ] Add robust error handling and retry mechanisms for Google Drive API & yt-dlp failures.
- [ ] Implement logging (e.g., `logrus` or `zap`) to monitor background task health.

---

## 🌿 Branching Strategy

Untuk menjaga keteraturan proses development dan menghindari konflik antar phase, kita menggunakan strategi branching sebagai berikut:

| Phase           | Branch Name                  | Deskripsi                                                       |
| --------------- | ---------------------------- | --------------------------------------------------------------- |
| **Production**  | `main`                       | Code yang sudah stabil dan siap digunakan.                      |
| **Integration** | `develop`                    | Branch utama untuk penggabungan tiap phase.                     |
| **Phase 1**     | `phase/01-foundation`        | Inisialisasi project, struktur Clean Architecture, & DB Schema. |
| **Phase 2**     | `phase/02-core-engine`       | Integrasi Spotify API, yt-dlp execution, & ID3 Tagging.         |
| **Phase 3**     | `phase/03-cloud-integration` | Integrasi Google Drive API & pembersihan file lokal.            |
| **Phase 4**     | `phase/04-api-construction`  | Pembangunan REST endpoints (Sync, Tracks, Status).              |
| **Phase 5**     | `phase/05-optimization`      | Worker Pool, robust logging, & error handling.                  |

### 🛠️ Development Workflow:

1. **Branching:** Selalu buat branch baru dari `develop` untuk memulai phase baru.
   ```bash
   git checkout develop
   git checkout -b phase/0x-nama-phase
   ```
2. **Isolation:** Fokus pada task yang ada di Roadmap phase tersebut. Jika ada error, perbaiki di branch phase tersebut sebelum di-merge.
3. **Merging:** Setelah phase selesai dan dites, lakukan merge ke `develop`.
   ```bash
   git checkout develop
   git merge phase/0x-nama-phase
   ```
4. **Update:** Selalu tarik (pull) perubahan terbaru dari `develop` sebelum memulai phase berikutnya.

---

## ⚙️ Prerequisites

To run this project locally, you need to install:

1. **Golang** (v1.25.0)
2. **Python** (v3.10.6) - *Required by yt-dlp*
3. **FFmpeg** (v4.4 or newer)
4. **yt-dlp** (latest version)
5. **PostgreSQL** (v14.5)
6. **Google Service Account Key** (JSON file placed in the root directory)
7. **Spotify Developer App** (Client ID & Client Secret)
8. **YouTube Cookies** (`cookies.txt` for yt-dlp rate-limit bypass)

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
```
