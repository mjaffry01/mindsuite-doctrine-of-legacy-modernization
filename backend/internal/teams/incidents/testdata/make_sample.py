"""Makes testdata/sample for the incidents & changes team test (run from this folder: python make_sample.py).

ServiceNow incident, change_request and problem list exports plus a Jira CSV export for one system (HarvestLink),
January to December 2025. Planted facts the test checks:
- month-end settlement failures (8, P1/P2, Settlement) without a problem record;
- label printer failures at intake (5) linked to problem PRB0001001;
- grower portal slowness (4, from the Jira export);
- 30 changes: 6 emergency, 5 unsuccessful, 2 with issues; CHG0030010 caused an incident;
- people's names in caller/assignee columns and in one title, an e-mail in a title, a password in one title.
"""
import csv, os, random
from datetime import datetime, timedelta

random.seed(7)
OUT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "sample")
os.makedirs(OUT, exist_ok=True)
F = "%Y-%m-%d %H:%M:%S"
people = ["Jane Smith", "Ravi Patel", "Tom Okafor", "Lena Fischer", "Ana Souza"]

inc = []
n = 1000


def add(title, ci, pri, opened, hours, cat="software", problem="", caused="", reopen=0, state="Closed"):
    global n
    n += 1
    resolved = (opened + timedelta(hours=hours)).strftime(F) if hours else ""
    inc.append({"number": f"INC00{n}", "short_description": title, "cmdb_ci": ci, "category": cat, "priority": pri, "state": state,
                "opened_at": opened.strftime(F), "resolved_at": resolved, "caller_id": random.choice(people), "assigned_to": random.choice(people),
                "problem_id": problem, "caused_by": caused, "reopen_count": reopen, "description": "Long text with customer details that must not be read"})


# month-end settlement failures
for i, m in enumerate([1, 2, 3, 4, 6, 8, 10, 11]):
    d = datetime(2025, m + 1, 1, 6, 30) - timedelta(days=1 if i % 2 else 0)
    add(f"Settlement batch failed at month end run {4400 + i}", "HarvestLink Settlement", "1 - Critical" if i < 3 else "2 - High", d, 9 + i, reopen=1 if i == 2 else 0)
# label printers at intake, with a problem record
for i, d in enumerate([datetime(2025, 2, 11, 8), datetime(2025, 4, 15, 9), datetime(2025, 6, 17, 7), datetime(2025, 9, 9, 8), datetime(2025, 11, 12, 10)]):
    add("Label printer not printing at intake scale", "Intake Scales", "3 - Moderate", d, 3 + i, cat="hardware", problem="PRB0001001")
# incident caused by a change (same component, caused_by)
add("Grading report shows wrong moisture after release", "HarvestLink Grading", "2 - High", datetime(2025, 3, 29, 8), 20, caused="CHG0030010")
# names, e-mail and a password in titles
add("Jane Smith cannot open grading report", "HarvestLink Grading", "4 - Low", datetime(2025, 5, 6, 11), 2)
add("Mailbox jane.smith@harvest.example full for exports", "Email", "4 - Low", datetime(2025, 7, 8, 9), 5)
add("Reset done, temporary password=Harvest2025x given to grower", "Grower Portal", "4 - Low", datetime(2025, 7, 22, 9), 1)
# background noise
words = ["Invoice PDF missing logo", "Scale reading delayed", "VPN to bank dropped", "Report export timeout", "Truck booking duplicate",
         "Moisture sensor offline", "Contract upload rejected", "Weighbridge camera blurry", "Payment file rejected by bank format", "Dashboard tile empty"]
for i, w in enumerate(words * 2):
    d = datetime(2024, 12, 16) + timedelta(days=17 * i + 3, hours=i % 9)
    add(w, random.choice(["HarvestLink Settlement", "HarvestLink Grading", "Intake Scales", "Grower Portal"]), random.choice(["3 - Moderate", "4 - Low"]), d, 4 + i % 30)
# an old open incident
add("Archive job leaves temp files", "HarvestLink Settlement", "3 - Moderate", datetime(2025, 9, 1, 10), 0, state="In Progress")

with open(os.path.join(OUT, "incident.csv"), "w", newline="", encoding="utf8") as fh:
    w = csv.DictWriter(fh, fieldnames=list(inc[0].keys()))
    w.writeheader()
    w.writerows(inc)

chg = []
for i in range(30):
    num = f"CHG00{30001 + i}"
    start = datetime(2025, 1, 4) + timedelta(days=12 * i)
    if i == 9:
        start = datetime(2025, 3, 28, 22)
    typ = "Emergency" if i % 5 == 0 else ("Standard" if i % 3 == 0 else "Normal")
    close = "Unsuccessful" if i in (2, 9, 14, 21, 27) else ("Successful with issues" if i in (6, 18) else "Successful")
    ci = ["HarvestLink Settlement", "HarvestLink Grading", "Grower Portal", "Intake Scales"][i % 4] if i != 9 else "HarvestLink Grading"
    chg.append({"number": num, "short_description": f"Release {i + 1} for {ci}", "type": typ, "cmdb_ci": ci, "state": "Closed", "close_code": close,
                "start_date": start.strftime(F), "end_date": (start + timedelta(hours=3)).strftime(F), "requested_by": random.choice(people),
                "assigned_to": random.choice(people), "backout_plan": "Restore the previous build"})
with open(os.path.join(OUT, "change_request.csv"), "w", newline="", encoding="utf8") as fh:
    w = csv.DictWriter(fh, fieldnames=list(chg[0].keys()))
    w.writeheader()
    w.writerows(chg)

with open(os.path.join(OUT, "problem.csv"), "w", newline="", encoding="utf8") as fh:
    w = csv.writer(fh)
    w.writerow(["number", "short_description", "cmdb_ci", "state", "known_error", "opened_at", "assigned_to"])
    w.writerow(["PRB0001001", "Label printer driver loses connection after sleep", "Intake Scales", "Root Cause Analysis", "true", "2025-04-16 10:00:00", "Tom Okafor"])
    w.writerow(["PRB0001002", "Bank payment file format changes yearly", "HarvestLink Settlement", "Closed", "true", "2025-02-01 10:00:00", "Ana Souza"])

with open(os.path.join(OUT, "jira-grower-portal.csv"), "w", newline="", encoding="utf8") as fh:
    w = csv.writer(fh)
    w.writerow(["Summary", "Issue key", "Issue Type", "Status", "Priority", "Component/s", "Reporter", "Assignee", "Created", "Resolved"])
    for i, d in enumerate([datetime(2025, 3, 3, 9, 15), datetime(2025, 5, 12, 14, 0), datetime(2025, 8, 4, 9, 40), datetime(2025, 12, 1, 8, 5)]):
        r = d + timedelta(hours=6 + i)
        w.writerow(["Grower portal login very slow", f"OPS-{120 + i}", "Incident", "Done", "High", "Grower Portal", "Lena Fischer", "Ravi Patel",
                    d.strftime("%d/%b/%y %I:%M %p").replace(" 0", " "), r.strftime("%d/%b/%y %I:%M %p").replace(" 0", " ")])

print("written to", OUT)
