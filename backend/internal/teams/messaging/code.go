package messaging

import (
	"path"
	"regexp"
	"sort"
	"strings"
)

// Messaging seen in the programs themselves: which program opens, reads or writes queues. Broker exports
// say what exists; the code says who uses it. For mainframe code this is often the only evidence given.

// CodeMQ is one program's use of a messaging API.
type CodeMQ struct {
	Tech, Program, File string
	Line                int
	Calls               []string
	Triggered           bool // started by an MQ trigger message (CICS / IMS trigger monitor)
}

const mqVerbs = `MQOPEN|MQGET|MQPUT1?|MQCLOSE|MQCONNX?|MQINQ|MQSET|MQCMIT|MQBACK|MQSUB`

var (
	// COBOL calls the MQ API by a quoted name (CALL 'MQGET' USING ...); PL/I and C call it like a
	// function (CALL MQGET(...); MQGET(hConn, ...)).
	mqCallRe    = regexp.MustCompile(`(?i)\bCALL\s+['"](` + mqVerbs + `)['"]|\b(` + mqVerbs + `)\s*\(`)
	mqTriggerRe = regexp.MustCompile(`(?i)\bMQTM\b|MQTM-QNAME|CMQTML`)
)

// mqInCode lists, per program, the MQ API calls in COBOL, PL/I and C code.
func mqInCode(r *Run) []*CodeMQ {
	var out []*CodeMQ
	for _, f := range r.Code.Files {
		if f.Lang != "COBOL" && f.Lang != "PL/I" && f.Lang != "C" && f.Lang != "C++" {
			continue
		}
		if ext := strings.ToLower(path.Ext(f.Path)); ext == ".h" || ext == ".hpp" {
			continue // headers declare the MQ API (cmqc.h); they do not call it
		}
		var u *CodeMQ
		for i, l := range f.Lines {
			if t := strings.TrimSpace(l); strings.HasPrefix(t, "*") || (len(l) > 6 && l[6] == '*') {
				continue
			}
			if m := mqCallRe.FindStringSubmatch(l); m != nil {
				if u == nil {
					u = &CodeMQ{Tech: "IBM MQ", Program: strings.ToUpper(strings.TrimSuffix(path.Base(f.Path), path.Ext(f.Path))), File: f.Path, Line: i + 1}
				}
				c := strings.ToUpper(m[1] + m[2])
				if !contains(u.Calls, c) {
					u.Calls = append(u.Calls, c)
				}
			}
			if mqTriggerRe.MatchString(l) && u != nil {
				u.Triggered = true
			}
		}
		if u != nil {
			if !u.Triggered {
				for _, l := range f.Lines {
					if mqTriggerRe.MatchString(l) {
						u.Triggered = true
						break
					}
				}
			}
			out = append(out, u)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Program < out[j].Program })
	return out
}

// addCodeMQ records one finding per program that uses a messaging API.
func addCodeMQ(r *Run) {
	for _, u := range mqInCode(r) {
		f := r.Code.File(u.File)
		if f == nil {
			continue
		}
		reads := contains(u.Calls, "MQGET")
		writes := contains(u.Calls, "MQPUT") || contains(u.Calls, "MQPUT1")
		role := "uses queues"
		switch {
		case reads && writes:
			role = "reads requests and sends replies"
		case reads:
			role = "reads messages"
		case writes:
			role = "sends messages"
		}
		detail := "Calls " + strings.Join(u.Calls, ", ") + "."
		if u.Triggered {
			detail += " It is started by an MQ trigger message, and the queue names come from that message at run time, so they are not in the code: take them from the queue manager (runmqsc DISPLAY QLOCAL, PROCESS)."
		}
		r.Store.Add(Finding{Agent: "flw", Kind: "code-messaging", Module: f.Module, Title: u.Program + " " + role + " (" + u.Tech + ")", Detail: detail,
			Citations: []Citation{cite(f, u.Line)}, Confidence: "high", Data: map[string]any{"program": u.Program, "calls": u.Calls, "tech": u.Tech, "triggered": u.Triggered}})
		r.Cover("flw", f)
	}
}
