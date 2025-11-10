# CI/CD Pipeline Monitoring Dashboard

Dashboard monitoring untuk CI/CD pipeline yang terintegrasi dengan GitHub Actions. Dashboard ini menampilkan status workflow runs dari semua repository dalam sebuah GitHub organization.


<img width="2108" height="1449" alt="image" src="https://github.com/user-attachments/assets/e25ee819-3187-409a-917b-d9bdba8be1b4" />

## Fitur

- 📊 **Dashboard Overview**: Statistik lengkap untuk Success, Failed, Running, Pending, dan Total Jobs
- 🔍 **Filter & Search**: Filter berdasarkan status dan pencarian berdasarkan nama job
- 🔄 **Auto Refresh**: Auto refresh setiap 30 detik untuk update data real-time
- 📱 **Responsive Design**: Tampilan yang responsif untuk berbagai ukuran layar
- 🎨 **Modern UI**: Interface yang bersih dan modern

## Teknologi

- **Backend**: Go dengan GitHub API integration
- **Frontend**: HTML, CSS, JavaScript (Vanilla)
- **API**: GitHub Actions API

## Prerequisites

- Go 1.21 atau lebih baru
- GitHub Personal Access Token dengan akses ke organization
- Akses ke repository dalam organization

## Setup

1. **Clone repository**

   ```bash
   git clone <repository-url>
   cd monitoring-cicd
   ```

2. **Install dependencies**

   ```bash
   go mod download
   ```

3. **Setup environment variables**

   Copy file `.env.example` ke `.env` dan edit dengan informasi Anda:

   ```bash
   cp .env.example .env
   ```

   Edit file `.env`:

   ```
   GITHUB_TOKEN=your_github_personal_access_token
   GITHUB_ORG=your_organization_name
   PORT=8080
   ```

   **Untuk Multiple Organizations:**

   Pisahkan nama organization dengan koma:

   ```
   GITHUB_ORG=org1,org2,org3
   ```

   **Atau** set environment variables secara manual:

   **Linux/Mac:**

   ```bash
   export GITHUB_TOKEN=your_github_personal_access_token
   export GITHUB_ORG=your_organization_name
   # Untuk multiple organizations:
   export GITHUB_ORG=org1,org2,org3
   export PORT=8080
   ```

   **Windows PowerShell:**

   ```powershell
   $env:GITHUB_TOKEN="your_github_personal_access_token"
   $env:GITHUB_ORG="your_organization_name"
   # Untuk multiple organizations:
   $env:GITHUB_ORG="org1,org2,org3"
   $env:PORT="8080"
   ```

   **Windows CMD:**

   ```cmd
   set GITHUB_TOKEN=your_github_personal_access_token
   set GITHUB_ORG=your_organization_name
   REM Untuk multiple organizations:
   set GITHUB_ORG=org1,org2,org3
   set PORT=8080
   ```

4. **Run aplikasi**

   ```bash
   go run main.go
   ```

5. **Akses dashboard**

   Buka browser dan akses: `http://localhost:8080`

## GitHub Token Setup

Untuk membuat GitHub Personal Access Token dengan permission yang benar:

### Cara Membuat Token:

1. **Buka GitHub Settings**

   - Login ke GitHub
   - Klik profile picture → **Settings**
   - Atau langsung ke: https://github.com/settings/profile

2. **Masuk ke Developer Settings**

   - Scroll ke bawah, klik **Developer settings**
   - Atau langsung ke: https://github.com/settings/apps

3. **Buat Personal Access Token**

   - Klik **Personal access tokens** → **Tokens (classic)**
   - Atau langsung ke: https://github.com/settings/tokens
   - Klik **Generate new token** → **Generate new token (classic)**

4. **Berikan Nama Token**

   - Contoh: `CI/CD Monitoring Dashboard`

5. **Pilih Expiration**

   - Pilih berapa lama token berlaku (No expiration untuk development, atau set sesuai kebutuhan)

6. **Pilih Permissions (SCOPE) yang DIPERLUKAN:**

   **⚠️ PENTING: Pilih permission berikut:**

   - ✅ **`repo`** (Full control of private repositories)

     - Ini memberikan akses ke semua repository dalam organization
     - Termasuk akses ke Actions API

   - ✅ **`read:org`** (Read org and team membership)

     - Untuk membaca informasi organization

   - ✅ **`workflow`** (Update GitHub Action workflows)
     - Untuk membaca workflow runs dan status

   **Catatan:**

   - Jika menggunakan **Fine-grained tokens** (token baru), pastikan:
     - Repository access: **All repositories** atau pilih repository yang ingin di-monitor
     - Permissions → Actions: **Read access**
     - Permissions → Metadata: **Read access** (otomatis)
     - Permissions → Contents: **Read access** (jika perlu)

7. **Generate Token**

   - Klik **Generate token** di bagian bawah
   - **⚠️ PENTING: Copy token SEKARANG!** Token hanya ditampilkan sekali
   - Simpan token di tempat yang aman

8. **Gunakan Token**
   - Copy token ke file `.env` sebagai `GITHUB_TOKEN`
   - Atau set sebagai environment variable

### Troubleshooting Permission:

Jika masih mendapat error 403:

1. **Pastikan token memiliki scope `repo`**

   - Token harus memiliki akses ke repository
   - Untuk organization, pastikan user memiliki akses ke repository

2. **Pastikan token memiliki scope `workflow`**

   - Ini diperlukan untuk membaca workflow runs

3. **Cek Organization Settings**

   - Pastikan organization mengizinkan third-party access
   - Settings → Third-party access → Pastikan tidak ada pembatasan

4. **Cek Repository Settings**

   - Pastikan repository tidak private atau token memiliki akses
   - Untuk private repository, token harus memiliki scope `repo`

5. **Coba Token di Browser**
   - Test dengan: `https://api.github.com/repos/ORGANIZATION/REPO/actions/runs`
   - Tambahkan header: `Authorization: token YOUR_TOKEN`
   - Jika masih 403, berarti token tidak memiliki permission yang cukup

## GitHub API Rate Limits

### Rate Limit untuk Free Tier:

GitHub API memiliki rate limit untuk mencegah abuse. Untuk **unauthenticated requests** dan **personal access tokens**:

- **5,000 requests per hour** untuk authenticated requests (dengan token)
- **60 requests per hour** untuk unauthenticated requests (tanpa token)

### Rate Limit untuk Actions API:

- **1,000 requests per hour** untuk Actions API endpoints
- Rate limit di-reset setiap jam

### Monitoring Rate Limit:

Aplikasi ini akan menampilkan rate limit information di log:

```
✅ Found 10 repositories in organization HRMS-OLIN
   Rate limit: 4990/5000 remaining (resets at 2025-11-10 10:00:00)
```

### Tips untuk Menghindari Rate Limit:

1. **Gunakan Personal Access Token**

   - Dengan token, Anda mendapat 5,000 requests/hour (vs 60 tanpa token)

2. **Monitor Rate Limit di Log**

   - Aplikasi menampilkan rate limit remaining di setiap request
   - Perhatikan log untuk melihat berapa banyak requests tersisa

3. **Kurangi Frekuensi Auto Refresh**

   - Jika menggunakan auto refresh, set interval lebih lama (misalnya 5 menit)
   - Atau matikan auto refresh jika tidak diperlukan

4. **Filter Repository**

   - Jika organization memiliki banyak repository, pertimbangkan untuk filter repository tertentu saja
   - Atau batasi jumlah workflow runs yang di-fetch per repository

5. **Caching**
   - Data di-cache di frontend, jadi refresh manual tidak akan selalu hit API
   - Auto refresh akan hit API setiap kali

### Jika Rate Limit Terlampaui:

Jika rate limit terlampaui, Anda akan mendapat error:

```
403 API rate limit exceeded
```

Solusi:

- Tunggu sampai rate limit reset (biasanya 1 jam)
- Atau gunakan GitHub Enterprise (jika tersedia) yang memiliki rate limit lebih tinggi

## Struktur Project

```
monitoring-cicd/
├── main.go              # Backend Go dengan GitHub API integration
├── go.mod               # Go module dependencies
├── static/              # Frontend files
│   ├── index.html      # HTML dashboard
│   ├── styles.css      # CSS styling
│   └── script.js       # JavaScript untuk interactivity
└── README.md           # Dokumentasi
```

## API Endpoints

### GET `/api/dashboard`

Mengembalikan data dashboard dengan statistik dan daftar jobs.

**Response:**

```json
{
  "stats": {
    "success": 53,
    "failed": 46,
    "running": 11,
    "pending": 40,
    "total": 150
  },
  "jobs": [
    {
      "id": "JOB-000001",
      "name": "Code Quality #1",
      "status": "success",
      "pipeline": "Backend CI",
      "branch": "release/v1.0",
      "duration": "27m 26s",
      "started": "1 day ago",
      "run_id": 123456789
    }
  ]
}
```

## Fitur Dashboard

### Filter & Search

- **Filter by Status**: Filter jobs berdasarkan status (Success, Failed, Running, Pending)
- **Search**: Pencarian berdasarkan nama job, ID, atau pipeline
- **Items per page**: Kontrol jumlah items per halaman (25, 50, 100)

### Auto Refresh

- Toggle auto refresh untuk update data otomatis setiap 30 detik
- Manual refresh dengan tombol Refresh

### Pagination

- Navigasi halaman untuk jobs yang banyak
- Kontrol jumlah items per halaman

## Development

### Menjalankan di development mode

```bash
go run main.go
```

### Build untuk production

```bash
go build -o monitoring-cicd main.go
./monitoring-cicd
```

## Troubleshooting

### Error: GITHUB_TOKEN environment variable is required

- Pastikan environment variable `GITHUB_TOKEN` sudah di-set
- Atau export di terminal sebelum menjalankan aplikasi

### Error: GITHUB_ORG environment variable is required

- Pastikan environment variable `GITHUB_ORG` sudah di-set dengan nama organization yang benar

### Tidak ada data yang muncul

- Pastikan GitHub token memiliki akses ke organization
- Pastikan ada workflow runs di repository dalam organization
- Check console browser untuk error messages


## Star History

[![Star History Chart](https://api.star-history.com/svg?repos=pharmaniaga/monitoring-cicd&type=Date)](https://star-history.com/#pharmaniaga/monitoring-cicd&Date)