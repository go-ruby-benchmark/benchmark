package benchmark

import "strings"

// Job is one labelled block awaiting measurement, as collected by Bmbm. It
// mirrors Benchmark::Job's (label, block) pairs and tracks the widest label.
type Job struct {
	width int
	list  []jobItem
}

type jobItem struct {
	label string
	fn    func()
}

// NewJob returns a Job with the given initial label-offset width, mirroring
// Benchmark::Job.new(width).
func NewJob(width int) *Job { return &Job{width: width} }

// Item registers a labelled block, widening the recorded label width, mirroring
// Benchmark::Job#item / #report.
func (j *Job) Item(label string, fn func()) *Job {
	if w := len(label); w > j.width {
		j.width = w
	}
	j.list = append(j.list, jobItem{label, fn})
	return j
}

// Width returns the widest label length seen.
func (j *Job) Width() int { return j.width }

// Benchmark is the general driver behind Bm: it prints caption, runs each block
// registered through the report builder, and appends summary lines built from
// the Tms slice the builder returns. It mirrors Benchmark.benchmark, returning
// the rendered table and the collected measurements.
//
// labelWidth and the per-line format match the Ruby parameters; the report's
// effective offset is labelWidth+1 (MRI's `label_width += 1`). build receives a
// *Report whose Record both times (via clock) and renders each row, and may
// return extra Tms summary rows; extraLabels override their labels in order.
func Benchmark(clock Clock, caption string, labelWidth int, format string, extraLabels []string, build func(report *Report) []Tms) (string, []Tms) {
	width := labelWidth + 1
	if format == "" {
		format = FORMAT
	}
	r := NewReport(clock, width, format)

	var sb strings.Builder
	extras := build(r)

	if caption != "" {
		sb.WriteString(strings.Repeat(" ", r.width))
		sb.WriteString(caption)
	}
	for _, t := range r.list {
		sb.WriteString(r.Line(t))
	}
	labels := append([]string(nil), extraLabels...)
	for _, t := range extras {
		name := ""
		if len(labels) > 0 {
			name, labels = labels[0], labels[1:]
		} else {
			name = t.label
		}
		sb.WriteString(ljust(name, width))
		sb.WriteString(t.Format(format))
	}
	return sb.String(), r.list
}

// Bm renders the simple bm table: CAPTION on top, one timed line per report
// item, with labels left-justified to labelWidth. extraLabels label any summary
// Tms the builder returns. It mirrors Benchmark.bm.
func Bm(clock Clock, labelWidth int, extraLabels []string, build func(report *Report) []Tms) (string, []Tms) {
	return Benchmark(clock, CAPTION, labelWidth, FORMAT, extraLabels, build)
}

// Bmbm renders the rehearsal-then-real bmbm report and returns the rendered text
// plus the real-run measurements. It mirrors Benchmark.bmbm: it computes the
// label width from the registered jobs (job width + 1), prints the rehearsal
// banner, a rehearsal line per job, the rehearsal total footer, then the take
// caption and a real line per job.
//
// Each block is measured twice with the injected clock (rehearsal then take), so
// a scripted clock makes the whole report deterministic.
func Bmbm(clock Clock, job *Job) (string, []Tms) {
	width := job.width + 1
	var sb strings.Builder

	// Rehearsal pass.
	sb.WriteString(RehearsalHeader(width))
	sb.WriteString("\n")
	total := Tms{}
	for _, it := range job.list {
		sb.WriteString(ljust(it.label, width))
		res := MeasureWith(clock, "", it.fn)
		sb.WriteString(res.Format(""))
		total = total.Add(res)
	}
	sb.WriteString(RehearsalFooter(width, total))

	// Take pass.
	sb.WriteString(TakeCaption(width))
	take := make([]Tms, 0, len(job.list))
	for _, it := range job.list {
		sb.WriteString(ljust(it.label, width))
		res := MeasureWith(clock, it.label, it.fn)
		sb.WriteString(res.Format(""))
		take = append(take, res)
	}
	return sb.String(), take
}
