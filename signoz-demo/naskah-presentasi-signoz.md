# Naskah Presentasi Screencast — SigNoz APM Demo

Gunakan naskah ini sebagai panduan saat merekam video. Sesuaikan dengan gaya bicara kamu sendiri.

---

## Bagian 1: Pembukaan (±30 detik)

> "Halo, perkenalkan saya [NAMA]. Di video ini saya akan mendemonstrasikan pemahaman
> Application Performance Monitoring menggunakan SigNoz, khususnya empat golden signals:
> RPS, Latency, Error Rate, dan Distributed Tracing.
>
> Saya menggunakan aplikasi demo berupa dua microservice Go: checkout-service yang memanggil
> payment-service. Keduanya sudah di-instrumentasi dengan OpenTelemetry dan mengirim data ke SigNoz."

---

## Bagian 2: Arsitektur (±1–2 menit)

> "Arsitekturnya sederhana: load generator menembak endpoint /checkout di checkout-service.
> Checkout-service lalu memanggil payment-service untuk memproses pembayaran.
>
> Yang penting di sini: karena kita pakai OpenTelemetry HTTP instrumentation, trace context
> diteruskan lewat header HTTP. Jadi satu request dari user akan menghasilkan satu trace
> yang melintasi kedua service — inilah yang disebut distributed tracing.
>
> Payment-service punya endpoint /config yang bisa kita pakai untuk mensimulasikan kondisi
> degraded: menaikkan latency atau error rate tanpa restart aplikasi."

*[Tunjukkan diagram di README atau gambar sederhana]*

---

## Bagian 3: Demo RPS (±1–2 menit)

> "Pertama, golden signal RPS — Requests Per Second. Ini mengukur throughput aplikasi,
> berapa banyak request yang kita handle per detik."

```bash
./load-generator.sh 10 120
```

> "Saya jalankan load generator sekitar 10 request per detik. Sekarang kita lihat di SigNoz,
> menu Services, pilih checkout-service. Grafik Rate naik — ini RPS kita.
>
> RPS penting untuk capacity planning. Kalau RPS terus naik tapi latency ikut naik,
> itu tanda kita perlu scale up atau optimasi."

*[Tunjukkan grafik Rate di SigNoz]*

---

## Bagian 4: Demo Latency (±2 menit)

> "Kedua, Latency — seberapa cepat aplikasi merespons. Di SigNoz kita bisa lihat
> percentile: p50, p90, dan p99."

```bash
curl -X POST "http://localhost:8082/config?latency_ms=800&error_rate=0"
```

> "Saya tambahkan 800 milidetik latency artificial di payment-service. Lihat grafik latency
> payment-service — p50 dan p99 ikut naik.
>
> p50 atau median artinya separuh request lebih cepat dari angka ini.
> p99 artinya 99% request lebih cepat — ini yang penting karena menangkap request paling lambat.
> User yang kebetulan kena request lambat di p99 akan merasa aplikasi 'lemot',
> walau rata-rata kelihatan normal."

*[Tunjukkan grafik latency sebelum dan sesudah]*

```bash
curl -X POST "http://localhost:8082/config?latency_ms=0&error_rate=0"
```

---

## Bagian 5: Demo Error Rate (±1–2 menit)

> "Ketiga, Error Rate — persentase request yang gagal."

```bash
curl -X POST "http://localhost:8082/config?latency_ms=0&error_rate=0.4"
```

> "Saya set error rate 40%. Artinya kira-kira 4 dari 10 request akan return HTTP 500.
> Di SigNoz, error rate payment-service naik. Trace yang error ditandai dengan status error —
> kita bisa filter trace yang gagal untuk investigasi."

*[Tunjukkan error rate chart dan satu error trace]*

---

## Bagian 6: Distributed Tracing (±2–3 menit)

> "Keempat, Distributed Tracing. Ini kekuatan utama APM — melacak perjalanan satu request
> dari service ke service."

*[Buka SigNoz → Traces → klik satu trace]*

> "Ini satu trace dari request /checkout. Flame graph-nya menunjukkan:
> - Span pertama: GET /checkout di checkout-service
> - Span kedua: GET /process di payment-service
> - Span ketiga: payment.process — custom span untuk logika bisnis
> - Span keempat: db.query — simulasi query database
>
> Dengan flame graph ini, kita bisa langsung lihat bottleneck ada di mana.
> Misalnya kalau db.query paling lebar, berarti masalahnya di database.
> Kalau payment.process yang lebar setelah kita naikkan latency_ms,
> itu sesuai ekspektasi karena delay ada di sana.
>
> Di production, ini sangat berguna saat incident: kita bisa trace satu request yang error
> dan lihat di service mana request itu gagal atau lambat."

---

## Bagian 7: Dashboard (±1–2 menit)

> "Ini dashboard yang saya buat sendiri di SigNoz. Saya susun panel-panel ini:
> - Request rate checkout-service untuk monitor throughput
> - Latency percentile payment-service
> - Error rate
> - Daftar trace error terbaru
>
> Saya kelompokkan per service supaya mudah dibaca saat on-call monitoring."

*[Scroll dashboard buatan sendiri]*

---

## Bagian 8: Alert (Bonus, ±1–2 menit)

> "(Opsional bonus) Saya juga konfigurasi alert di SigNoz. Rule-nya: error rate payment-service
> di atas 20% selama 5 menit."

```bash
curl -X POST "http://localhost:8082/config?latency_ms=0&error_rate=0.9"
./load-generator.sh 5 60
```

> "Saya trigger error rate tinggi... dan notifikasi masuk ke [Telegram/Teams].
> Ini memungkinkan proactive monitoring — kita tahu ada masalah sebelum user komplain."

*[Tunjukkan notifikasi]*

---

## Bagian 9: Penutup (±30 detik)

> "Kesimpulannya, dengan SigNoz dan OpenTelemetry kita bisa:
> - Monitor throughput lewat RPS
> - Deteksi perlambatan lewat latency percentile
> - Track kegagalan lewat error rate
> - Investigasi root cause lewat distributed tracing
>
> Terima kasih sudah menonton."
