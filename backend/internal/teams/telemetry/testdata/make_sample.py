"""Deterministic telemetry sample: 12 months of hourly Splunk counts with a harvest season, an NGINX log and a little code."""
import os, datetime as dt
D = os.path.join(os.path.dirname(os.path.abspath(__file__)), 'sample') + '/'
os.makedirs(D + 'app', exist_ok=True)

# Splunk timechart span=1h count, errors, perc95(duration): 2025-11-01 .. 2026-10-31, missing 10 days in February
rows = ['_time,count,errors,perc95(duration)']
t = dt.datetime(2025, 11, 1)
end = dt.datetime(2026, 11, 1)
while t < end:
    if not (t.month == 2 and 5 <= t.day <= 14):
        h, wd = t.hour, t.weekday()  # Monday = 0
        base = 40 if 7 <= h <= 17 else 6
        if 8 <= h <= 10:
            base = 90
        if wd >= 5:
            base = base // 3
        if wd == 6 and 1 <= h <= 4:
            base = 0
        if t.month in (8, 9, 10):
            base *= 2
        if t == dt.datetime(2026, 9, 14, 9):
            base = 1800  # busiest hour of the year
        errors = base // 100
        p95 = 850 if 8 <= h <= 10 else 420
        rows.append(f'{t:%Y-%m-%dT%H:%M:%S.000+00:00},{base},{errors},{p95}')
    t += dt.timedelta(hours=1)
open(D + 'splunk-requests-per-hour.csv', 'w', newline='').write('\n'.join(rows) + '\n')

# NGINX access log with $request_time, a few hours on 2026-09-15
lines = []
t0 = dt.datetime(2026, 9, 15, 8, 0, 0)
reqs = []
for i in range(120):
    reqs.append(('POST', '/api/deliveries', 201, 0.180))
for i in range(80):
    reqs.append(('GET', f'/api/lots/{1000 + i}?token=Zx81abc', 200, 0.090))
for i in range(30):
    reqs.append(('GET', f'/api/growers/grower{i}@example.com/statement', 200, 0.300))
for i in range(25):
    reqs.append(('GET', '/api/reports/yearly', 200, 2.600))
for i in range(40):
    reqs.append(('POST', '/api/payments/run', 500 if i % 3 == 0 else 200, 0.700))
for i in range(60):
    reqs.append(('GET', '/static/app.js', 200, 0.004))
reqs.append(('GET', '/api/admin/legacy-export', 200, 1.200))
for i, (m, u, st, rt) in enumerate(reqs):
    ts = t0 + dt.timedelta(seconds=i * 37)
    ip = f'10.0.{i % 3}.{(i * 7) % 40 + 1}'
    lines.append(f'{ip} - - [{ts:%d/%b/%Y:%H:%M:%S} +0000] "{m} {u} HTTP/1.1" {st} 512 "-" "Mozilla/5.0" {rt:.3f}')
open(D + 'nginx-access.log.txt', 'w', newline='').write('\n'.join(lines) + '\n')

open(D + 'app/LotController.java', 'w', newline='').write('''package com.harvest.lots;

@RestController
@RequestMapping("/api/lots")
public class LotController {
    @GetMapping("/{id}")
    Lot get(@PathVariable long id) { return null; }

    @GetMapping("/archive")
    List<Lot> archive() { return null; }
}
''')
open(D + 'app/ExportController.java', 'w', newline='').write('''package com.harvest.export;

@RestController
public class ExportController {
    @GetMapping("/api/export/csv")
    String csv() { return ""; }

    @PostMapping("/api/deliveries")
    void deliver() { }
}
''')
open(D + 'app/monitoring.properties', 'w', newline='').write('splunk.url=https://splunk.harvest.local:8089\nsplunk.password=SplunkPa55word!\n')
print('sample ok', len(rows), len(lines))
