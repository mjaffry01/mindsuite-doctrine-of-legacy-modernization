package documents

// Short names for the shared core (internal/agents), so this team's agents read like the rest of the backend.

import "github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/agents"

type (
	Run       = agents.Run
	File      = agents.File
	Finding   = agents.Finding
	Citation  = agents.Citation
	Reading   = agents.Reading
	Sent      = agents.Sent
	AgentInfo = agents.AgentInfo
)

const (
	BasisCode      = agents.BasisCode
	BasisInferred  = agents.BasisInferred
	BasisModel     = agents.BasisModel
	BasisHistory   = agents.BasisHistory
	StatusProposed = agents.StatusProposed
	StatusAccepted = agents.StatusAccepted
	StatusRejected = agents.StatusRejected
	StatusConflict = agents.StatusConflict
	MsgQuestion    = agents.MsgQuestion
	MsgHandoff     = agents.MsgHandoff
)

var (
	ModelStrong   = agents.ModelStrong
	ModelBalanced = agents.ModelBalanced
	CodeLangs     = agents.CodeLangs
	LegacyLangs   = agents.LegacyLangs
	ruleLikeRe    = agents.RuleLikeRe
	reportSchema  = agents.ReportSchema
	ev            = agents.Arr(agents.Str())
)

func cite(f *File, n int) Citation               { return agents.Cite(f, n) }
func looksSecret(l string) bool                  { return agents.LooksSecret(l) }
func lineOf(f *File, offset int) int             { return agents.LineOf(f, offset) }
func short(s string, n int) string               { return agents.Short(s, n) }
func appendUnique(s []string, v string) []string { return agents.AppendUnique(s, v) }
func contains(s []string, v string) bool         { return agents.Contains(s, v) }
func uniq(s []string) []string                   { return agents.Uniq(s) }
func toInt(v any) int                            { return agents.ToInt(v) }
func toStrings(v any) []string                   { return agents.ToStrings(v) }
func firstN(s []string, n int) []string          { return agents.FirstN(s, n) }
func sortedKeys[V any](m map[string]V) []string  { return agents.SortedKeys(m) }
func obj(props map[string]any) map[string]any    { return agents.Obj(props) }
func arr(items map[string]any) map[string]any    { return agents.Arr(items) }
func str() map[string]any                        { return agents.Str() }
func integer() map[string]any                    { return agents.Integer() }
func enum(v ...string) map[string]any            { return agents.Enum(v...) }
func eachLine(r *Run, agent string, keep func(*File) bool, fn func(f *File, n int, line string)) {
	agents.EachLine(r, agent, keep, fn)
}
