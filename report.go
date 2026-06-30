package benchmark

import "strings"

// Report accumulates the measurements taken inside a bm/benchmark block and
// renders the table. It mirrors Benchmark::Report: a label offset (width) and a
// per-line format, growing the offset to fit the widest label, with the same
// off-by-one as MRI (the caller adds one to label_width before constructing it).
type Report struct {
	width  int
	format string
	list   []Tms
	clock  Clock
}

// NewReport returns a Report with the given label offset and per-line format,
// timing its blocks with clock. An empty format selects FORMAT, matching
// Benchmark::Report.new(width, format) where a nil format defers to Tms#format's
// default.
func NewReport(clock Clock, width int, format string) *Report {
	return &Report{width: width, format: format, clock: clock}
}

// Width returns the current label offset.
func (r *Report) Width() int { return r.width }

// Format returns the per-line format string (empty means the FORMAT default).
func (r *Report) Format() string { return r.format }

// List returns the measurements collected so far.
func (r *Report) List() []Tms { return r.list }

// Run times fn with the report's clock under the given label, appends the
// measurement (widening the label offset to fit a longer label), and returns the
// resulting Tms. It mirrors Benchmark::Report#item / #report, which call
// Benchmark.measure(label, &blk) and stash the result.
func (r *Report) Run(label string, fn func()) Tms {
	if w := len(label); w > r.width {
		r.width = w
	}
	t := MeasureWith(r.clock, label, fn)
	r.list = append(r.list, t)
	return t
}

// Line renders one report row for t: the label left-justified to the report
// offset, then the formatted time columns. This is the pure rendering MRI does
// in Benchmark#benchmark's per-item loop.
func (r *Report) Line(t Tms) string {
	return ljust(t.label, r.width) + t.Format(r.format)
}

// Caption returns the heading line for the report: width leading spaces followed
// by CAPTION, exactly as Benchmark#benchmark prints it (" "*report.width +
// caption).
func (r *Report) Caption() string {
	return strings.Repeat(" ", r.width) + CAPTION
}

// ExtraLine renders a trailing summary row built from a Tms the block returned
// (e.g. a total or average). label overrides the Tms label when non-empty,
// matching `(labels.shift || t.label || "").ljust(label_width)` in
// Benchmark#benchmark; labelWidth is the original label_width+1 offset.
func (r *Report) ExtraLine(labelWidth int, label string, t Tms) string {
	name := label
	if name == "" {
		name = t.label
	}
	return ljust(name, labelWidth) + t.Format(r.format)
}

// RehearsalHeader returns the bmbm rehearsal banner line, matching
// 'Rehearsal '.ljust(width+CAPTION.length, '-'), where width is the bmbm label
// offset (job width + 1).
func RehearsalHeader(width int) string {
	return ljustFill("Rehearsal ", width+len(CAPTION), '-')
}

// RehearsalFooter returns the bmbm rehearsal total line, matching
// " #{ets}\n\n".rjust(width+CAPTION.length+2, '-') where ets is the summed Tms
// formatted with "total: %tsec".
func RehearsalFooter(width int, total Tms) string {
	ets := total.Format("total: %tsec")
	return rjustFill(" "+ets+"\n\n", width+len(CAPTION)+2, '-')
}

// TakeCaption returns the bmbm "take" (real run) caption line, matching
// ' '*width + CAPTION.
func TakeCaption(width int) string {
	return strings.Repeat(" ", width) + CAPTION
}

// ljustFill left-justifies s to width, padding on the right with pad, like
// Ruby's String#ljust(width, padstr) for a single-character pad.
func ljustFill(s string, width int, pad byte) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(string(pad), width-len(s))
}

// rjustFill right-justifies s to width, padding on the left with pad, like
// Ruby's String#rjust(width, padstr) for a single-character pad.
func rjustFill(s string, width int, pad byte) string {
	if len(s) >= width {
		return s
	}
	return strings.Repeat(string(pad), width-len(s)) + s
}
