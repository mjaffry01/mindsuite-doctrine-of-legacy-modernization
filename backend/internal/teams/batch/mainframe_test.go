package batch

import (
	"strings"
	"testing"

	"github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/agents"
)

func mainframeRun(t *testing.T, files map[string]string) *agents.Run {
	t.Helper()
	b := map[string][]byte{}
	for k, v := range files {
		b[k] = []byte(v)
	}
	r, err := agents.NewRun(agents.LoadFiles("mf", b), agents.Options{Team: "batch"}, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// TestMainframeBatch reads a Control-M XML export, a CA-7 listing and the JCL members.
func TestMainframeBatch(t *testing.T) {
	r := mainframeRun(t, map[string]string{
		"ctm/deftable.xml": `<?xml version="1.0" encoding="UTF-8"?>
<DEFTABLE>
 <FOLDER FOLDER_NAME="FIN_MONTHEND" DATACENTER="DC1">
  <JOB JOBNAME="GLEXTR" MEMNAME="GLEXTR" MEMLIB="PROD.JCLLIB" TASKTYPE="Job" NODEID="MVSA" DAYS="ALL" TIMEFROM="0130" DESCRIPTION="GL extract">
   <OUTCOND NAME="GLEXTR-OK" SIGN="+"/>
  </JOB>
  <JOB JOBNAME="GLPOST" MEMNAME="GLPOST" TASKTYPE="Job" WEEKDAYS="1,2,3,4,5">
   <INCOND NAME="GLEXTR-OK"/>
   <INCOND NAME="NEVER-ADDED"/>
  </JOB>
 </FOLDER>
</DEFTABLE>
`,
		"ca7/ljob.txt": ` LJOB,JOB=PAYD100,LIST=ALL
 JOB      ----JCL----  SYSTEM  USR MCNT
 PAYD100  PAYD100      PAYROLL 001 0001
 ---------------- TRIGGERED JOBS ----------------
    JOB=PAYD200  SCHID=001
`,
		"jcl/PAYD100.jcl": `//PAYD100  JOB (ACCT),'PAYROLL',CLASS=A
//* run the payroll calculation
//STEP01   EXEC PGM=PAYCALC
//STEP02   EXEC PGM=IKJEFT01
`,
		"jcl/GLEXTR.jcl": `//GLEXTR   JOB (ACCT),'GL',CLASS=A
//STEP01   EXEC PGM=GLX001
`,
		"jcl/ORPHAN.jcl": `//ORPHAN   JOB (ACCT),'OLD',CLASS=A
//STEP01   EXEC PROC=OLDPROC
`,
		"jcl/MYPROC.jcl": `//MYPROC   PROC
//STEP01   EXEC PGM=NOTAJOB
`,
	})
	m := ParseBatch(r)
	get := func(name string) *Job {
		t.Helper()
		j := m.Jobs[strings.ToLower(name)]
		if j == nil {
			t.Fatalf("job %s not found; have %v", name, sortedKeys(m.Jobs))
		}
		return j
	}

	// Control-M XML
	gl := get("GLEXTR")
	if gl.Scheduler != "Control-M" || gl.Folder != "FIN_MONTHEND" || gl.Start != "01:30" || gl.Host != "MVSA" {
		t.Errorf("GLEXTR: %+v", gl)
	}
	if !strings.Contains(gl.Schedule, "every day") || !strings.Contains(gl.Schedule, "from 01:30") || !gl.MonthEnd {
		t.Errorf("GLEXTR schedule %q month-end %v", gl.Schedule, gl.MonthEnd)
	}
	if gl.Command != "JCL member GLEXTR in PROD.JCLLIB runs GLX001" {
		t.Errorf("GLEXTR command %q", gl.Command)
	}
	if post := get("GLPOST"); len(post.After) != 1 || post.After[0] != "GLEXTR" || post.Schedule != "weekdays 1,2,3,4,5 (folder FIN_MONTHEND)" {
		t.Errorf("GLPOST after %v schedule %q", post.After, post.Schedule)
	}
	if len(m.MissingEvents) != 1 || !strings.Contains(m.MissingEvents[0], "NEVER-ADDED") {
		t.Errorf("missing events %v", m.MissingEvents)
	}

	// CA-7
	pay := get("PAYD100")
	if pay.Scheduler != "CA-7" || pay.Command != "JCL member PAYD100 runs PAYCALC, IKJEFT01" {
		t.Errorf("PAYD100: scheduler %q command %q", pay.Scheduler, pay.Command)
	}
	if pay.File != "ca7/ljob.txt" || pay.CmdLine != 0 {
		t.Errorf("PAYD100 should keep the CA-7 listing as its source: %s line %d", pay.File, pay.CmdLine)
	}
	if t2 := get("PAYD200"); len(t2.After) != 1 || t2.After[0] != "PAYD100" || !strings.Contains(t2.Schedule, "triggered by PAYD100") {
		t.Errorf("PAYD200 after %v schedule %q", t2.After, t2.Schedule)
	}

	// JCL only
	if o := get("ORPHAN"); !strings.HasPrefix(o.Scheduler, "JCL only") || o.Command != "runs OLDPROC" || o.CmdLine != 2 {
		t.Errorf("ORPHAN: %+v", o)
	}
	if m.Jobs["myproc"] != nil {
		t.Error("a PROC member is not a job")
	}
}
