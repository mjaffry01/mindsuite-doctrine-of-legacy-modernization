package messaging

import (
	"reflect"
	"testing"

	"github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/agents"
)

// TestMQInCode finds IBM MQ calls in COBOL, PL/I and C programs.
func TestMQInCode(t *testing.T) {
	files := map[string][]byte{
		"cobol/MQREAD.cbl": []byte(`       PROCEDURE DIVISION.
           MOVE MQTM-QNAME TO WS-QNAME.
           CALL 'MQOPEN' USING HCONN OBJDESC OPTS HOBJ CC RC.
           CALL 'MQGET'  USING HCONN HOBJ MSGDESC GMO LEN BUF DLEN CC RC.
      *    CALL 'MQCLOSE' USING HCONN HOBJ OPTS CC RC.
           CALL "MQPUT1" USING HCONN OBJDESC MSGDESC PMO LEN BUF CC RC.
`),
		"pli/MQSEND.pli": []byte(` MQSEND: PROC OPTIONS(MAIN);
   CALL MQPUT(HCONN, HOBJ, MSGDESC, PMO, LEN, BUF, CC, RC);
 END MQSEND;
`),
		"c/mqpoll.c": []byte(`#include <cmqc.h>
int main(void) {
  MQCONN(qm, &hConn, &cc, &rc);
  MQGET (hConn, hObj, &md, &gmo, len, buf, &dlen, &cc, &rc);
}
`),
		"c/cmqc.h": []byte(`void MQENTRY MQGET (MQHCONN Hconn, MQHOBJ Hobj);
`),
		"cobol/PLAIN.cbl": []byte(`       PROCEDURE DIVISION.
           CALL 'CUSTUPD' USING WS-REC.
`),
	}
	r, err := agents.NewRun(agents.LoadFiles("mf", files), agents.Options{Team: "messaging"}, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]*CodeMQ{}
	for _, u := range mqInCode(r) {
		got[u.Program] = u
	}
	want := map[string]struct {
		calls     []string
		triggered bool
	}{
		"MQREAD": {[]string{"MQOPEN", "MQGET", "MQPUT1"}, true},
		"MQSEND": {[]string{"MQPUT"}, false},
		"MQPOLL": {[]string{"MQCONN", "MQGET"}, false},
	}
	if len(got) != len(want) {
		t.Errorf("programs %v, want %d (headers and plain programs excluded)", keysOf(got), len(want))
	}
	for p, w := range want {
		u := got[p]
		if u == nil {
			t.Errorf("%s not found", p)
			continue
		}
		if !reflect.DeepEqual(u.Calls, w.calls) || u.Triggered != w.triggered {
			t.Errorf("%s: calls %v triggered %v, want %v %v", p, u.Calls, u.Triggered, w.calls, w.triggered)
		}
	}
}

func keysOf(m map[string]*CodeMQ) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}
