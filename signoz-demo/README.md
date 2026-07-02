# SigNoz APM Demo — Panduan Lengkap

Project ini terdiri dari 2 service Go (`checkout-service` → `payment-service`) yang
sudah di-instrumentasi dengan **OpenTelemetry** dan mengirim traces ke **SigNoz**.
Tujuannya: mendemonstrasikan 4 golden signals (RPS, Latency, Error Rate, Distributed Tracing)
secara live dan terkontrol — cocok untuk hands-on assignment L1 Monitoring.

> Adaptasi dari `datadog-demo` dengan stack yang sama (2 microservice + chaos simulator),
> namun exporter diganti ke OTLP/SigNoz.

---

## 1. Arsitektur

```
load-generator.sh ──> checkout-service (:8081) ──> payment-service (:8082)
                              │                              │
                              └──────── trace context ───────┘
                                   (1 trace, 2 services)
                                          │
                                          v
                              SigNoz OTLP Collector (:4318 HTTP)
                                          │
                                          v
                                   SigNoz UI (DRC / lokal)
```

| Komponen | Peran |
|----------|-------|
| `checkout-service` | Entry point — mensimulasikan API gateway |
| `payment-service` | Service "berat" + endpoint `/config` untuk ubah latency & error rate live |
| `load-generator.sh` | Menembak traffic agar grafik RPS terlihat jelas di SigNoz |
| OpenTelemetry | Auto-instrumentasi HTTP + custom span (`payment.process`, `db.query`) |

Karena trace context diteruskan lewat HTTP header (W3C Trace Context via `otelhttp`),
satu request `/checkout` menghasilkan **satu trace** yang melintasi dua service —
inilah distributed tracing yang akan kamu jelaskan di screencast.

---

## 2. Cara Menjalankan

### Prasyarat

- Docker & Docker Compose
- Akses ke SigNoz (dashboard DRC tim, atau SigNoz lokal di `:4318`)

### Langkah

```bash
cd signoz-demo
cp .env.example .env
# Edit .env — isi OTEL_EXPORTER_OTLP_ENDPOINT sesuai URL SigNoz kamu
# Contoh DRC: http://<host-signoz-drc>:4318
# Contoh lokal: http://host.docker.internal:4318

docker compose up --build
```

Cek service hidup:

```bash
curl http://localhost:8081/healthz   # checkout-service
curl http://localhost:8082/healthz   # payment-service
curl http://localhost:8081/checkout  # 1 request penuh lintas 2 service
```

Tunggu ~30 detik, lalu buka SigNoz → **Services** — kamu harus melihat
`checkout-service` dan `payment-service`.

### Jalankan tanpa Docker (opsional)

```bash
# Terminal 1 — payment-service
cd service-b
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
go run .

# Terminal 2 — checkout-service
cd service-a
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
export PAYMENT_SERVICE_URL=http://localhost:8082
go run .
```

---

## 3. Demo 4 Golden Signals 

### a. RPS (Requests Per Second)

```bash
chmod +x load-generator.sh
./load-generator.sh 10 180   # ~10 req/detik selama 3 menit
```

Buka **SigNoz → Services → checkout-service**, lihat grafik **Rate (ops/s)** naik
real-time. Coba ubah angka RPS (`5` lalu `20`) supaya grafik kelihatan jelas
saat kamu jelaskan di video.

**Konsep untuk dijelaskan:** RPS mengukur throughput — berapa banyak request
yang diproses per detik. Naiknya RPS tanpa scaling = indikator beban tinggi.

### b. Latency (p50 / p95 / p99)

Sambil traffic jalan, naikkan latency `payment-service`:

```bash
curl -X POST "http://localhost:8082/config?latency_ms=800&error_rate=0"
```

Lihat di **SigNoz → Services → payment-service**, grafik Latency (p50, p90, p99)
ikut naik. Balikin ke normal:

```bash
curl -X POST "http://localhost:8082/config?latency_ms=0&error_rate=0"
```

**Konsep untuk dijelaskan:**
- **p50 (median):** separuh request lebih cepat dari ini
- **p95/p99:** "ekor" distribusi — request paling lambat; sering jadi indikator
  masalah nyata walau rata-rata kelihatan baik-baik saja

### c. Error Rate

```bash
curl -X POST "http://localhost:8082/config?latency_ms=0&error_rate=0.4"
```

Ini bikin ~40% request gagal (HTTP 500). Lihat **Error Rate** naik di
`payment-service`, dan trace yang gagal ditandai error di **Traces**.

**Konsep untuk dijelaskan:** Error rate = persentase request yang gagal.
Korelasi dengan RPS: error rate tinggi saat traffic normal = bug/downstream issue.

### d. Distributed Tracing

Buka **SigNoz → Traces**, filter `service.name = checkout-service`, klik salah
satu trace. Kamu akan melihat flame graph:

```
checkout-service: GET /checkout
  └─ payment-service: GET /process
       └─ payment.process
            └─ db.query (simulasi)
```

Tunjukkan dan jelaskan flame graph di video — cara melacak bottleneck ada di span mana.

---

## 4. Membuat Dashboard SigNoz (30 poin — wajib individu)

Buka **Dashboards → New Dashboard**. Tambahkan panel berikut:

| Panel | Query / Sumber | Tipe |
|-------|----------------|------|
| Request Rate (RPS) | Metrics → `signoz_calls_total` filter `service_name=checkout-service`, Rate | Time Series |
| Latency p50/p90/p99 | APM → Latency panel `payment-service` | Time Series |
| Error Rate (%) | Metrics → `signoz_calls_total` dengan `status_code=STATUS_CODE_ERROR` / total | Time Series |
| Service Overview | Services widget untuk kedua service | Value / Stat |
| Top Operations | Traces grouped by `name` | Table |
| Recent Error Traces | Traces filter `hasError=true` | List |

> **Catatan:** Nama metric di SigNoz bisa sedikit berbeda tergantung versi.
> Alternatif: buat panel dari halaman **Services** → klik service → **Add to Dashboard**.

Tips biar nilainya bagus:

- Kelompokkan panel per service (checkout vs payment) pakai **Row**
- Judul panel jelas, bukan default query string
- Tambahkan **variable** `service` di atas dashboard biar reusable
- Time range default: **Past 1 hour**

---

## 5. Endpoint Simulator (`/config`)

Ubah perilaku `payment-service` tanpa restart:

```bash
# Normal
curl -X POST "http://localhost:8082/config?latency_ms=0&error_rate=0"

# Latency tinggi (800ms tambahan per request)
curl -X POST "http://localhost:8082/config?latency_ms=800&error_rate=0"

# Error rate 40%
curl -X POST "http://localhost:8082/config?latency_ms=0&error_rate=0.4"

# Kombinasi (degraded state)
curl -X POST "http://localhost:8082/config?latency_ms=500&error_rate=0.3"
```

---
