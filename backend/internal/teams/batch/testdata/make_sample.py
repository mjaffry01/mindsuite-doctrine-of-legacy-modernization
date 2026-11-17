"""Deterministic batch sample: Control-M, AutoSys, cron and Windows Task Scheduler exports, 30 days of run history and one script."""
import os, json, datetime as dt
D = os.path.join(os.path.dirname(os.path.abspath(__file__)), 'sample') + '/'
os.makedirs(D + 'scripts', exist_ok=True)

ctm = {
    "HL_NIGHTLY": {
        "Type": "Folder", "ControlmServer": "ctm-prod",
        "EXTRACT_DELIVERIES": {"Type": "Job:Command", "Command": "/opt/hl/bin/extract_deliveries.sh", "Host": "batch-prd-01", "RunAs": "hlbatch",
                               "Description": "Extract the day's deliveries", "When": {"Schedule": "Everyday", "FromTime": "0200"},
                               "eventsToAdd": {"Type": "AddEvents", "Events": [{"Event": "EXTRACT-TO-CALC"}]}},
        "CALC_SETTLEMENT": {"Type": "Job:Command", "Command": "/opt/hl/bin/calc_settlement.sh", "Host": "batch-prd-01", "RunAs": "hlbatch",
                            "Description": "Calculate grower payments", "When": {"Schedule": "Everyday"},
                            "eventsToWaitFor": {"Type": "WaitForEvents", "Events": [{"Event": "EXTRACT-TO-CALC"}]},
                            "eventsToAdd": {"Type": "AddEvents", "Events": [{"Event": "CALC-TO-PAY"}, {"Event": "CALC-TO-REPORT"}]}},
        "SEND_PAYMENTS": {"Type": "Job:FileTransfer", "Command": "sftp -b /opt/hl/sftp/payments.batch bank@sftp.partnerbank.example", "Host": "batch-prd-01", "RunAs": "root",
                          "Description": "Send the payment file to the bank", "When": {"Schedule": "Everyday"},
                          "eventsToWaitFor": {"Type": "WaitForEvents", "Events": [{"Event": "CALC-TO-PAY"}]}},
        "PRINT_STATEMENTS": {"Type": "Job:Command", "Command": "/opt/hl/bin/print_statements.sh", "Host": "batch-prd-01", "RunAs": "hlbatch",
                             "When": {"Schedule": "Everyday"},
                             "eventsToWaitFor": {"Type": "WaitForEvents", "Events": [{"Event": "CALC-TO-REPORT"}]}},
        "MONTH_END_CLOSE": {"Type": "Job:Command", "Command": "/opt/hl/bin/month_end_close.sh", "Host": "batch-prd-01", "RunAs": "hlbatch",
                            "Description": "Close the month", "When": {"Schedule": "LastDayOfMonth", "FromTime": "0400"},
                            "eventsToWaitFor": {"Type": "WaitForEvents", "Events": [{"Event": "GL_FEED-TO-CLOSE"}]}},
    }
}
open(D + 'controlm-jobs.json', 'w', newline='').write(json.dumps(ctm, indent=2) + '\n')

open(D + 'autosys_jobs.jil', 'w', newline='').write('''/* ----------------- HL_REPORTS ----------------- */
insert_job: HL_REPORTS   job_type: BOX
owner: hlbatch@app-prod-01
description: "Daily reports"

insert_job: HL_LOAD   job_type: CMD
box_name: HL_REPORTS
command: /opt/hl/bin/load_reports.sh
machine: app-prod-01
owner: hlbatch@app-prod-01
start_times: "05:00"

insert_job: HL_DAILY_REPORT   job_type: CMD
box_name: HL_REPORTS
command: /opt/hl/bin/daily_report.sh
machine: app-prod-01
owner: hlbatch@app-prod-01
condition: s(HL_LOAD)
start_times: "10:00"

insert_job: HL_OLD_EXPORT   job_type: CMD
command: /opt/hl/bin/old_export.sh
machine: app-prod-01
owner: hlbatch@app-prod-01
status: ON_ICE
''')

open(D + 'crontab-batch01-appuser.txt', 'w', newline='').write('''# m h dom mon dow command
MAILTO=ops@harvest.example
*/5 * * * * /opt/hl/bin/poll_files.sh
0 1 28-31 * * /opt/hl/bin/month_end.sh
# 0 3 * * * /opt/hl/bin/old_cleanup.sh
30 2 * * * sqlplus hl/Tiger123@HLPROD @/opt/hl/sql/purge.sql
''')

open(D + 'schtasks.csv', 'w', newline='').write('''"HostName","TaskName","Next Run Time","Status","Logon Mode","Last Run Time","Last Result","Author","Task To Run","Start In","Comment","Scheduled Task State","Run As User","Schedule Type","Start Time","Days"
"WIN-BATCH-01","\\HL\\Archive Scans","18/11/2026 23:00:00","Ready","Interactive/Background","17/11/2026 23:00:00","0","HL","C:\\HL\\archive_scans.ps1","C:\\HL","Archive scanned documents","Enabled","SYSTEM","Daily","23:00:00","Every 1 day(s)"
"WIN-BATCH-01","\\Microsoft\\Windows\\Defrag\\ScheduledDefrag","N/A","Ready","Background","N/A","0","Microsoft","defrag.exe","","","Enabled","SYSTEM","Weekly","01:00:00","SUN"
''')

rows = ['job,start,end,status']
day = dt.datetime(2026, 10, 1)
for d in range(30):
    t = day + dt.timedelta(days=d)
    rows.append(f'EXTRACT_DELIVERIES,{t:%Y-%m-%d} 02:00:00,{t:%Y-%m-%d} 02:20:00,ENDED_OK')
    mins = 60 + d * 2  # grows from 60 to 118 minutes
    s = t.replace(hour=2, minute=20)
    if d % 6 == 5:  # fails every sixth day, rerun 40 minutes later
        rows.append(f'CALC_SETTLEMENT,{s:%Y-%m-%d %H:%M:%S},{s + dt.timedelta(minutes=15):%Y-%m-%d %H:%M:%S},ENDED_NOTOK')
        s = s + dt.timedelta(minutes=40)
    rows.append(f'CALC_SETTLEMENT,{s:%Y-%m-%d %H:%M:%S},{s + dt.timedelta(minutes=mins):%Y-%m-%d %H:%M:%S},ENDED_OK')
    p = s + dt.timedelta(minutes=mins)
    rows.append(f'SEND_PAYMENTS,{p:%Y-%m-%d %H:%M:%S},{p + dt.timedelta(minutes=5):%Y-%m-%d %H:%M:%S},ENDED_OK')
    rows.append(f'PRINT_STATEMENTS,{p:%Y-%m-%d %H:%M:%S},{p + dt.timedelta(minutes=30):%Y-%m-%d %H:%M:%S},ENDED_OK')
    rows.append(f'HL_LOAD,{t:%Y-%m-%d} 05:00:00,{t:%Y-%m-%d} 05:10:00,SUCCESS')
    rows.append(f'HL_DAILY_REPORT,{t:%Y-%m-%d} 10:00:00,{t:%Y-%m-%d} 10:05:00,SUCCESS')
open(D + 'run-history.csv', 'w', newline='').write('\n'.join(rows) + '\n')

open(D + 'scripts/poll_files.sh', 'w', newline='').write('#!/bin/sh\n# move new scale files into the intake folder\nfor f in /data/in/*.csv; do mv "$f" /data/intake/; done\n')
print('sample ok', len(rows))
