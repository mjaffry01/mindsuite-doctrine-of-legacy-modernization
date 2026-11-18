package database

import (
	"reflect"
	"testing"

	"github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/agents"
)

// TestEmbeddedSQL lists COBOL programs with EXEC SQL and the tables they name.
func TestEmbeddedSQL(t *testing.T) {
	files := map[string][]byte{
		"src/CUSTUPD.cbl": []byte(`       IDENTIFICATION DIVISION.
       PROGRAM-ID. CUSTUPD.
      * EXEC SQL SELECT X FROM COMMENTED END-EXEC.
       PROCEDURE DIVISION.
           EXEC SQL
             SELECT NAME INTO :WS-NAME
               FROM PROD.CUSTOMER
              WHERE ID = :WS-ID
           END-EXEC.
           EXEC SQL UPDATE ACCOUNTS SET BAL = 0 END-EXEC.
           EXEC SQL INCLUDE SQLCA END-EXEC.
           MOVE 'FROM NOWHERE' TO WS-MSG.
`),
		"src/NOSQL.cbl": []byte(`       PROCEDURE DIVISION.
           DISPLAY 'HELLO'.
`),
		"jcl/BIND.jcl": []byte(`//BIND     JOB (ACCT)
//STEP1    EXEC PGM=IKJEFT01
//SYSTSIN  DD *
  DSN SYSTEM(DB2P)
  BIND PACKAGE(COLL) MEMBER(CUSTUPD)
`),
	}
	r, err := agents.NewRun(agents.LoadFiles("mf", files), agents.Options{Team: "database"}, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	m := ParseDB(r)
	if len(m.EmbeddedSQL) != 1 {
		t.Fatalf("want 1 program with embedded SQL, got %+v", m.EmbeddedSQL)
	}
	e := m.EmbeddedSQL[0]
	if e.Program != "CUSTUPD" || e.File != "src/CUSTUPD.cbl" || e.Line != 5 || e.Statements != 3 {
		t.Errorf("program %+v", e)
	}
	if want := []string{"PROD.CUSTOMER", "ACCOUNTS"}; !reflect.DeepEqual(e.Tables, want) {
		t.Errorf("tables %v, want %v", e.Tables, want)
	}
}
