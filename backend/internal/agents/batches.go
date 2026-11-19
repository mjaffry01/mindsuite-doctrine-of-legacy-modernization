package agents

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const batchTargetLines = 5000
const batchMinimumLines = 3000
const batchTargetBytes = 16 << 20
const batchVersion = 1

type BatchProgress struct {
	Total     int    `json:"total"`
	Completed int    `json:"completed"`
	Reused    int    `json:"reused"`
	Failed    int    `json:"failed"`
	Oversized int    `json:"oversized"`
	Phase     string `json:"phase"`
}

type batchFile struct {
	file               *File
	start, end, header int
}

type evidenceBatch struct {
	group        string
	files        []batchFile
	lines, bytes int
}

var declarationBoundary = regexp.MustCompile("(?i)^\\s*(?:end\\s+(?:sub|function|property|class|module)|[}];?)\\s*$")

func cutEvidence(team string, f *File) []batchFile {
	if len(f.Lines) <= batchTargetLines {
		return []batchFile{{file: f, end: len(f.Lines)}}
	}
	split := func(n int) bool {
		line := strings.TrimSpace(f.Lines[n-1])
		if team == "code" && CodeLangs[f.Lang] {
			return declarationBoundary.MatchString(line) && (strings.HasPrefix(strings.ToLower(line), "end ") || (n < len(f.Lines) && strings.TrimSpace(f.Lines[n]) == ""))
		}
		if team == "documents" && f.Lang == "Document" {
			return line == ""
		}
		return false
	}
	ext := strings.ToLower(filepath.Ext(f.Path))
	rows, header := team == "telemetry" && ext == ".log", 0
	if rows {
		for _, line := range f.Lines {
			if strings.HasPrefix(strings.TrimSpace(line), "#Fields:") {
				rows = false
				break
			}
		}
	}
	if team == "telemetry" && ext == ".csv" && strings.Contains(f.Lines[0], ",") {
		rows, header = true, 1
		for _, line := range f.Lines {
			if strings.Count(line, "\\\"")%2 != 0 {
				rows, header = false, 0
				break
			}
		}
	}
	var out []batchFile
	start := 0
	for len(f.Lines)-start > batchTargetLines {
		end := start + batchTargetLines - header
		if !rows {
			for end >= start+batchMinimumLines && !split(end) {
				end--
			}
			if end < start+batchMinimumLines {
				break
			}
		}
		out = append(out, batchFile{file: f, start: start, end: end, header: headerForSegment(start, header)})
		start = end
	}
	return append(out, batchFile{file: f, start: start, end: len(f.Lines), header: headerForSegment(start, header)})

}

func headerForSegment(start, header int) int {
	if start == 0 {
		return 0
	}
	return header
}

func (part batchFile) originalLine(n int) int {
	if n <= part.header {
		return n
	}
	return part.start + n - part.header
}

func batchGroup(team string, f *File) string {
	if team == "code" && f.Module != "" {
		return f.Module
	}
	return strings.ToLower(filepath.Ext(f.Path))
}

func planEvidenceBatches(code *Codebase, team string) []evidenceBatch {
	if code.TotalLines() <= batchTargetLines && code.total <= batchTargetBytes {
		return nil
	}
	grouped := map[string][]*File{}
	for _, file := range code.Files {
		group := batchGroup(team, file)
		grouped[group] = append(grouped[group], file)
	}
	groups := make([]string, 0, len(grouped))
	for group := range grouped {
		groups = append(groups, group)
	}
	sort.Strings(groups)
	var batches []evidenceBatch
	for _, group := range groups {
		batch := evidenceBatch{group: group}
		flush := func() {
			if len(batch.files) > 0 {
				batches = append(batches, batch)
			}
			batch = evidenceBatch{group: group}
		}
		for _, file := range grouped[group] {
			fragments := cutEvidence(team, file)
			for _, part := range fragments {
				lines := part.end - part.start + part.header
				bytes := file.Bytes * lines / max(1, len(file.Lines))
				if len(batch.files) > 0 && (batch.lines+lines > batchTargetLines || batch.bytes+bytes > batchTargetBytes || len(fragments) > 1) {
					flush()
				}
				batch.files = append(batch.files, part)
				batch.lines += lines
				batch.bytes += bytes
				if len(fragments) > 1 || batch.lines >= batchTargetLines || batch.bytes >= batchTargetBytes {
					flush()
				}
			}
		}
		flush()
	}
	return batches
}

func (b evidenceBatch) codebase(base *Codebase) *Codebase {
	part := &Codebase{Root: base.Root, Branch: base.Branch, Commit: base.Commit, total: b.bytes, byPath: map[string]*File{}}
	for _, segment := range b.files {
		file := *segment.file
		file.Lines = file.Lines[segment.start:segment.end]
		if segment.header > 0 {
			file.Lines = append(append([]string(nil), segment.file.Lines[:segment.header]...), file.Lines...)
		}
		file.Bytes = segment.file.Bytes * len(file.Lines) / max(1, len(segment.file.Lines))
		part.Files = append(part.Files, &file)
		part.byPath[file.Path] = &file
	}
	return part
}

func batchSignature(b evidenceBatch, options Options, model bool) string {
	digest := sha256.New()
	version := batchVersion
	if options.Team == "code" || options.Team == "database" {
		version = 2
	} // Structured operating flows changed the cached Code batch findings.
	// Interview terms are hashed for cache invalidation and remain absent from checkpoint metadata.
	settings, _ := json.Marshal(struct {
		Version   int
		Team      string
		Budget    int
		Model     bool
		Tiers     []string
		Interview *InterviewFocus
	}{
		version, options.Team, options.MaxModelFiles, model, []string{ModelStrong, ModelBalanced, ModelFast}, options.Interview,
	})
	digest.Write(settings)
	for _, segment := range b.files {
		key, _ := json.Marshal([]any{segment.file.Path, segment.file.Module, segment.file.Lang, segment.start, segment.end, segment.header})
		digest.Write(key)
		for _, line := range segment.file.Lines[segment.start:segment.end] {
			digest.Write([]byte(line))
			digest.Write([]byte{'\n'})
		}
	}
	return hex.EncodeToString(digest.Sum(nil))
}

func checkpointPath(dir, signature string) string {
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, signature+".json")
}

func readBatchCheckpoint(name string) (Result, bool) {
	var result Result
	if name == "" {
		return result, false
	}
	data, err := os.ReadFile(name)
	if err != nil || json.Unmarshal(data, &result) != nil || result.Run == nil || result.Run.Status != "done" || len(result.Run.Problems) > 0 {
		return Result{}, false
	}
	return result, true
}

func saveBatchCheckpoint(name string, result Result) error {
	if name == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(name), 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(name), ".batch-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	encodeErr := json.NewEncoder(file).Encode(result)
	closeErr := file.Close()
	if err := errors.Join(encodeErr, closeErr); err != nil {
		return err
	}
	return os.Rename(file.Name(), name)
}

func (r *Run) setBatchProgress(progress BatchProgress) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Batches = &progress
}

func (r *Run) executeBatches(ctx context.Context, batches []evidenceBatch) {
	progress := BatchProgress{Total: len(batches), Phase: "reviewing"}
	for _, batch := range batches {
		if batch.lines > batchTargetLines || batch.bytes > batchTargetBytes {
			progress.Oversized++
		}
	}
	r.setBatchProgress(progress)
	r.Step(1, "done", fmt.Sprintf("Indexed %d files into %d evidence batches", len(r.Code.Files), len(batches)))
	r.Step(2, "done", "Batches keep source and structured evidence boundaries")
	r.Step(3, "running", fmt.Sprintf("Reviewing %d batches", len(batches)))
	global, usesGlobalPass := r.team.(interface{ FinalizeBatches(context.Context, *Run) })
	modelReady, _ := r.ModelReady()
	modelReady = modelReady && !usesGlobalPass
	if ranking, ok := r.team.(interface{ BatchPriority(*Run, []string) int }); ok && r.Options.Interview != nil {
		scores := map[*File]int{}
		score := func(batch evidenceBatch) int {
			points := 0
			for _, part := range batch.files {
				file := part.file
				if _, found := scores[file]; !found {
					scores[file] = ranking.BatchPriority(r, []string{file.Path, file.Module})
				}
				points = max(points, scores[file])
			}
			return points
		}
		sort.SliceStable(batches, func(i, j int) bool { return score(batches[i]) > score(batches[j]) })
	}
	budget := r.Options.MaxModelFiles
	sources := map[string][]Finding{}
	for i, batch := range batches {
		if ctx.Err() != nil {
			break
		}
		opts := r.Options
		opts.BatchDir = ""
		opts.GitHistory = false
		opts.AllowModel = modelReady && budget > 0
		if opts.AllowModel {
			opts.MaxModelFiles = min(budget, len(batch.files))
			budget -= opts.MaxModelFiles
		}
		name := checkpointPath(r.Options.BatchDir, batchSignature(batch, opts, opts.AllowModel))
		result, reused := readBatchCheckpoint(name)
		if reused {
			progress.Reused++
			for _, attempt := range result.Telemetry {
				attempt.Batch = i + 1
				attempt.Reused = true
				r.recordAttempt(attempt)
			}
		} else {
			child, err := NewRun(batch.codebase(r.Code), opts, r.LLM, "")
			if err != nil {
				r.Store.Log("orch", fmt.Sprintf("Batch %d could not start: %v", i+1, err))
				progress.Failed++
				r.setBatchProgress(progress)
				continue
			}
			child.attemptSink = func(attempt AgentAttempt) {
				attempt.Batch = i + 1
				r.recordAttempt(attempt)
			}
			child.executeSingle(ctx)
			result = child.Snapshot()
			if result.Run.Status == "done" && len(result.Run.Problems) == 0 {
				if err := saveBatchCheckpoint(name, result); err != nil {
					r.Store.Log("orch", fmt.Sprintf("Batch %d could not save progress: %v", i+1, err))
					progress.Failed++
					if !usesGlobalPass {
						r.addBatchResult(i+1, batch, result, sources)
					}
					r.setBatchProgress(progress)
					continue
				}
			}
		}
		if !usesGlobalPass {
			r.addBatchResult(i+1, batch, result, sources)
		}
		if result.Run.Status == "done" && len(result.Run.Problems) == 0 {
			progress.Completed++
		} else {
			progress.Failed++
			for _, failure := range result.Run.Problems {
				r.Store.Log(failure.Agent, fmt.Sprintf("Batch %d: %s", i+1, failure.Error))
			}
			if result.Run.Error != "" {
				r.Store.Log("orch", fmt.Sprintf("Batch %d: %s", i+1, result.Run.Error))
			}
		}
		r.setBatchProgress(progress)
		r.Store.Log("orch", fmt.Sprintf("Batches completed: %d/%d (%d reused, %d failed)", progress.Completed, progress.Total, progress.Reused, progress.Failed))
	}
	r.Step(3, "done", fmt.Sprintf("%d of %d batches completed", progress.Completed, progress.Total))
	if usesGlobalPass && progress.Completed == progress.Total {
		progress.Phase = "connecting"
		r.setBatchProgress(progress)
		r.Store.Log("orch", "Reconciling the complete telemetry timeline before computing global metrics.")
		global.FinalizeBatches(ctx, r)
		r.mu.Lock()
		r.Report += "\n" + batchCoverageSection(progress)
		r.mu.Unlock()
		progress.Phase = "complete"
		r.setBatchProgress(progress)
		return
	}

	progress.Phase = "connecting"
	r.setBatchProgress(progress)
	r.Step(4, "done", "Completed within the available evidence batches")
	if modelReady {
		r.Step(5, "done", "Used the review's selected-excerpt budget across batches")
	} else {
		r.Step(5, "skipped", "This batch pass used local rules")
	}
	r.Step(6, "running", "Connecting related findings across batches")
	r.reconcileBatchFindings(sources)
	if reconciler, ok := r.team.(interface {
		ReconcileBatches(*Run, func(Citation) int)
	}); ok && progress.Completed == progress.Total {
		type interval struct{ start, end, batch int }
		paths := map[string][]interval{}
		for i, batch := range batches {
			for _, part := range batch.files {
				paths[part.file.Path] = append(paths[part.file.Path], interval{part.start, part.end, i + 1})
			}
		}
		reconciler.ReconcileBatches(r, func(c Citation) int {
			for _, part := range paths[c.File] {
				if c.Line > part.start && c.Line <= part.end {
					return part.batch
				}
			}
			return 0
		})
	}
	r.Step(6, "done", "Cross-batch evidence linked")
	r.Step(7, "done", "Batch findings verified against their source")
	r.Step(8, "running", "Preparing coverage and combined evidence report")
	report := r.batchReport(progress)
	r.mu.Lock()
	r.Report = report
	if progress.Completed != progress.Total {
		r.Status = "failed"
		r.Error = "Some evidence batches are incomplete. Resume analysis to retry them."
		progress.Phase = "incomplete"
	} else {
		progress.Phase = "complete"
	}
	r.mu.Unlock()
	r.setBatchProgress(progress)
	r.Step(8, "done", "Combined report saved with batch coverage")
	r.Step(9, "people", "Confirm proposals and unresolved links with the project owner")
}

func (r *Run) addBatchResult(number int, batch evidenceBatch, result Result, sources map[string][]Finding) {
	prefix := fmt.Sprintf("b%d-", number)
	segments := map[string]batchFile{}
	for _, segment := range batch.files {
		segments[segment.file.Path] = segment
	}
	skipped := map[string]bool{}
	for _, file := range result.Coverage.NotRead {
		skipped[file.Path] = true
	}
	for _, segment := range batch.files {
		if !skipped[segment.file.Path] && result.Run.Status == "done" && result.Coverage.Files-result.Coverage.Analysed <= len(result.Coverage.NotRead) {
			r.Cover("batch", segment.file)
		}
	}
	for _, finding := range result.Findings {
		finding.ID = prefix + finding.ID
		for i := range finding.Evidence {
			finding.Evidence[i] = prefix + finding.Evidence[i]
		}
		remapped := map[string]bool{}
		for i, c := range finding.Citations {
			segment := segments[c.File]
			if segment.start > 0 {
				if c.Line > 0 {
					c.Line = segment.originalLine(c.Line)
				}
				if c.EndLine > 0 {
					c.EndLine = segment.originalLine(c.EndLine)
				}
				if !remapped[c.File] && finding.Detail != "" {
					pattern := regexp.MustCompile(regexp.QuoteMeta(c.File) + ":([0-9]+)")
					finding.Detail = pattern.ReplaceAllStringFunc(finding.Detail, func(location string) string {
						number, err := strconv.Atoi(strings.TrimPrefix(location, c.File+":"))
						if err != nil || number <= 0 {
							return location
						}
						return fmt.Sprintf("%s:%d", c.File, segment.originalLine(number))
					})
					remapped[c.File] = true
				}
			}
			finding.Citations[i] = c
		}
		r.Store.addImported(finding)
		if finding.Status == StatusAccepted && len(finding.Citations) > 0 {
			name := finding.Title
			if table, ok := finding.Data["table"].(string); ok && table != "" {
				name = table
			}
			key := finding.Kind + "\x00" + strings.ToLower(strings.TrimSpace(name))
			if len(sources[key]) < 20 {
				sources[key] = append(sources[key], finding)
			}
		}
	}
	for _, question := range result.Questions {
		if question.Ref != "" {
			question.Ref = prefix + question.Ref
		}
		r.Store.Question(question.From, question.Ref, question.Text)
	}
	r.mu.Lock()
	r.SentToModel = append(r.SentToModel, result.Sent...)
	r.mu.Unlock()
}

func (r *Run) reconcileBatchFindings(sources map[string][]Finding) {
	keys := make([]string, 0, len(sources))
	for key := range sources {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		findings := sources[key]
		batches := map[string]bool{}
		for _, finding := range findings {
			batch, _, _ := strings.Cut(finding.ID, "-")
			batches[batch] = true
		}
		if len(batches) < 2 {
			continue
		}
		ids := make([]string, 0, len(findings))
		for _, finding := range findings {
			ids = append(ids, finding.ID)
		}
		kind, name, _ := strings.Cut(key, "\x00")
		f := Finding{Agent: "orch", Kind: "cross-batch-link", Title: fmt.Sprintf("%s appears in %d batches", name, len(batches)), Detail: "Related " + kind + " evidence. Confirm whether these references describe the same entity.", Evidence: ids, Basis: BasisInferred, Status: StatusConflict}
		r.Store.Add(f)
	}
}

func batchCoverageSection(progress BatchProgress) string {
	return fmt.Sprintf("## Batch coverage\n\n%d of %d batches completed. %d reused saved results. %d failed. %d contained an item above the usual size target to preserve its source format.\n\n", progress.Completed, progress.Total, progress.Reused, progress.Failed, progress.Oversized)
}

func (r *Run) batchReport(progress BatchProgress) string {
	var report strings.Builder
	fmt.Fprintf(&report, "# %s analysis — %s\n\n", r.Options.Team, r.Code.Root)
	report.WriteString(batchCoverageSection(progress))
	fmt.Fprintf(&report, "The source loader supplied %d lines in %d files; %d files were skipped at load time. Batch estimates describe their own evidence. Use cross-batch links to review dependencies and overlapping records.\n\n", r.Code.TotalLines(), len(r.Code.Files), len(r.Code.Skipped))
	if progress.Completed != progress.Total {
		fmt.Fprint(&report, "Some evidence remains unreviewed. Resume analysis from the project evidence card.\n\n")
	}
	findings := r.Store.Findings()
	if contextual, ok := r.team.(interface{ BatchReviewContext(*Run) string }); ok {
		report.WriteString(contextual.BatchReviewContext(r))
	}
	for _, kind := range []string{"cross-batch-link", "workflow-trace", "business-rule", "purpose", "other"} {
		title := map[string]string{"cross-batch-link": "Cross-batch links", "workflow-trace": "Workflow traces", "business-rule": "Business rules", "purpose": "Business purpose", "other": "Other findings"}[kind]
		fmt.Fprintf(&report, "## %s\n\n", title)
		count, total := 0, 0
		for _, f := range findings {
			if f.Status == StatusRejected || f.Kind != kind && !(kind == "other" && f.Kind != "cross-batch-link" && f.Kind != "workflow-trace" && f.Kind != "business-rule" && f.Kind != "purpose") {
				continue
			}
			total++
			if count == 50 {
				continue
			}
			fmt.Fprintf(&report, "- %s [%s] (%s)", f.Title, f.ID, f.Status)
			if len(f.Citations) > 0 {
				fmt.Fprintf(&report, " — %s:%d", f.Citations[0].File, f.Citations[0].Line)
			}
			fmt.Fprintln(&report)
			if f.Detail != "" {
				fmt.Fprintf(&report, "  %s\n", f.Detail)
			}
			if kind == "cross-batch-link" && len(f.Evidence) > 0 {
				fmt.Fprintf(&report, "  Supporting findings: %s\n", strings.Join(f.Evidence, ", "))
			}
			count++
		}
		if total == 0 {
			fmt.Fprintln(&report, "No findings in this section.")
		}
		if total > count {
			fmt.Fprintf(&report, "%d additional findings in All findings.\n", total-count)
		}
		fmt.Fprintln(&report)
	}
	return report.String()
}
