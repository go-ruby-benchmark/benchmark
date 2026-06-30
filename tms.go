// Package benchmark is a pure-Go (no cgo) reimplementation of the deterministic,
// interpreter-independent core of Ruby's standard-library Benchmark module
// (MRI 4.0.5 / benchmark 0.5.0): the Tms measurement value type, its memberwise
// arithmetic and %-directive formatting, and the report layout (caption, label
// justification, the bm/bmbm tables) — all expressed as pure functions over Tms
// values.
//
// The only impure ingredient, the clock, is injected. MRI's Benchmark.measure
// reads Process.times (user/system CPU, plus children's) and a monotonic real
// clock; here a Clock seam supplies those numbers, so the host (go-embedded-ruby)
// wires in the real process clock while tests feed a fixed one for byte-for-byte
// deterministic output.
package benchmark

import (
	"fmt"
	"regexp"
	"strings"
)

// CAPTION is the default heading printed above a column of times. It matches
// Benchmark::Tms::CAPTION (and Benchmark::CAPTION) exactly, trailing newline
// included.
const CAPTION = "      user     system      total        real\n"

// FORMAT is the default per-line format string, matching Benchmark::Tms::FORMAT
// (and Benchmark::FORMAT). The %u/%y/%t/%r directives are the Benchmark
// extensions resolved by Tms.Format.
const FORMAT = "%10.6u %10.6y %10.6t %10.6r\n"

// BenchmarkVersion mirrors Benchmark::BENCHMARK_VERSION.
const BenchmarkVersion = "2002-04-25"

// Tms holds the times associated with one benchmark measurement, mirroring
// Benchmark::Tms: user CPU time, system CPU time, the children's user and system
// CPU times, the elapsed real time, and a label. Total is the sum of the four
// CPU times.
type Tms struct {
	utime  float64
	stime  float64
	cutime float64
	cstime float64
	real   float64
	total  float64
	label  string
}

// NewTms returns a Tms with the given times and label. As in MRI, total is
// computed as utime+stime+cutime+cstime and a nil-equivalent (empty) label is
// stored verbatim (MRI calls label.to_s, so nil becomes "").
func NewTms(utime, stime, cutime, cstime, real float64, label string) Tms {
	return Tms{
		utime:  utime,
		stime:  stime,
		cutime: cutime,
		cstime: cstime,
		real:   real,
		total:  utime + stime + cutime + cstime,
		label:  label,
	}
}

// Utime returns the user CPU time.
func (t Tms) Utime() float64 { return t.utime }

// Stime returns the system CPU time.
func (t Tms) Stime() float64 { return t.stime }

// Cutime returns the children's user CPU time.
func (t Tms) Cutime() float64 { return t.cutime }

// Cstime returns the children's system CPU time.
func (t Tms) Cstime() float64 { return t.cstime }

// Real returns the elapsed real (wall-clock) time.
func (t Tms) Real() float64 { return t.real }

// Total returns utime+stime+cutime+cstime, matching Tms#total.
func (t Tms) Total() float64 { return t.total }

// Label returns the measurement label.
func (t Tms) Label() string { return t.label }

// Add returns a new Tms whose times are the memberwise sum of t and other,
// matching Tms#+. The result label is empty, exactly as MRI's memberwise leaves
// it.
func (t Tms) Add(other Tms) Tms { return t.memberwiseTms(opAdd, other) }

// Sub returns the memberwise difference t-other, matching Tms#-.
func (t Tms) Sub(other Tms) Tms { return t.memberwiseTms(opSub, other) }

// Mul returns the memberwise product t*other, matching Tms#* with a Tms operand.
func (t Tms) Mul(other Tms) Tms { return t.memberwiseTms(opMul, other) }

// Div returns the memberwise quotient t/other, matching Tms#/ with a Tms operand.
func (t Tms) Div(other Tms) Tms { return t.memberwiseTms(opDiv, other) }

// AddScalar returns a new Tms with x added to every time, matching Tms#+ with a
// numeric operand (MRI applies the scalar to all five fields, real included).
func (t Tms) AddScalar(x float64) Tms { return t.memberwiseScalar(opAdd, x) }

// SubScalar returns a new Tms with x subtracted from every time.
func (t Tms) SubScalar(x float64) Tms { return t.memberwiseScalar(opSub, x) }

// MulScalar returns a new Tms with every time multiplied by x, matching Tms#*.
func (t Tms) MulScalar(x float64) Tms { return t.memberwiseScalar(opMul, x) }

// DivScalar returns a new Tms with every time divided by x, matching Tms#/.
func (t Tms) DivScalar(x float64) Tms { return t.memberwiseScalar(opDiv, x) }

type binop int

const (
	opAdd binop = iota
	opSub
	opMul
	opDiv
)

func apply(op binop, a, b float64) float64 {
	switch op {
	case opAdd:
		return a + b
	case opSub:
		return a - b
	case opMul:
		return a * b
	default: // opDiv
		return a / b
	}
}

func (t Tms) memberwiseTms(op binop, x Tms) Tms {
	return NewTms(
		apply(op, t.utime, x.utime),
		apply(op, t.stime, x.stime),
		apply(op, t.cutime, x.cutime),
		apply(op, t.cstime, x.cstime),
		apply(op, t.real, x.real),
		"",
	)
}

func (t Tms) memberwiseScalar(op binop, x float64) Tms {
	return NewTms(
		apply(op, t.utime, x),
		apply(op, t.stime, x),
		apply(op, t.cutime, x),
		apply(op, t.cstime, x),
		apply(op, t.real, x),
		"",
	)
}

// ToA returns the 6-element slice [label, utime, stime, cutime, cstime, real],
// mirroring Tms#to_a.
func (t Tms) ToA() []any {
	return []any{t.label, t.utime, t.stime, t.cutime, t.cstime, t.real}
}

// ToS formats t with the default FORMAT, mirroring Tms#to_s.
func (t Tms) ToS() string { return t.Format("") }

// directive matches a printf-style flag/width/precision run followed by one of
// the Benchmark extension letters. It is the Go analogue of MRI's
// /(%[-+.\d]*)X/ substitutions in Tms#format.
var directive = regexp.MustCompile(`%[-+.\d]*[nuyUYtr]`)

// Format renders t according to format, honouring the Benchmark extensions on
// top of standard %-directives, matching Tms#format byte-for-byte:
//
//	%n  label    %u  utime   %y  stime (system)   %U  cutime
//	%t  total    %r  real (wrapped in parentheses) %Y  cstime
//
// The flag/width/precision run before the extension letter is preserved (e.g.
// %10.6u). An empty format selects FORMAT, in which case the trailing args are
// ignored (as MRI does when format is nil). With an explicit format, any
// remaining standard directives are filled from args.
func (t Tms) Format(format string, args ...any) string {
	useDefault := format == ""
	src := format
	if useDefault {
		src = FORMAT
	}

	str := directive.ReplaceAllStringFunc(src, func(m string) string {
		// m is like "%10.6u": split the leading run from the trailing letter.
		run := m[:len(m)-1]
		letter := m[len(m)-1]
		switch letter {
		case 'n':
			return fmt.Sprintf(run+"s", t.label)
		case 'u':
			return fmt.Sprintf(run+"f", t.utime)
		case 'y':
			return fmt.Sprintf(run+"f", t.stime)
		case 'U':
			return fmt.Sprintf(run+"f", t.cutime)
		case 'Y':
			return fmt.Sprintf(run+"f", t.cstime)
		case 't':
			return fmt.Sprintf(run+"f", t.total)
		default: // 'r'
			return "(" + fmt.Sprintf(run+"f", t.real) + ")"
		}
	})

	if useDefault {
		// MRI: `format ? str % args : str` — with a nil format the substituted
		// string is returned verbatim (no `%`-pass, so `%%` is left intact only
		// because FORMAT contains none).
		return str
	}
	// With an explicit format, MRI always evaluates `str % args`. Ruby's
	// String#% collapses `%%`→`%` and silently ignores surplus args when no
	// directive consumes them; Go's fmt.Sprintf instead appends `%!(EXTRA …)`.
	// To stay byte-for-byte faithful we only forward args when a standard
	// directive remains to consume them, and always run the pass so `%%`
	// collapses.
	if remainingDirective.MatchString(str) {
		return fmt.Sprintf(str, args...)
	}
	// No consumable directive: a no-op pass that only collapses `%%`. Written
	// with an explicit empty args slice so `go vet`'s printf check stays quiet.
	return fmt.Sprintf(str, []any{}...)
}

// remainingDirective detects a standard printf directive left after the
// Benchmark-extension substitutions, i.e. a `%` run ending in a non-`%`
// conversion letter (so a literal `%%` does not count).
var remainingDirective = regexp.MustCompile(`%[-+ #0-9.*]*[a-zA-Z]`)

// ljust right-pads s with spaces to width, like Ruby's String#ljust. A string
// already at least width long is returned unchanged.
func ljust(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
