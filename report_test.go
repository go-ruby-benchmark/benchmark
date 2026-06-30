package benchmark

import (
	"reflect"
	"testing"
)

func TestBmLayout(t *testing.T) {
	c := newFixedClock()
	out, list := Bm(c, 7, nil, func(r *Report) []Tms {
		r.Run("for:", func() {})
		r.Run("times:", func() {})
		return nil
	})
	const want = "              user     system      total        real\n" +
		"for:      0.100000   0.050000   0.187500 (  1.000000)\n" +
		"times:    0.100000   0.050000   0.187500 (  1.000000)\n"
	if out != want {
		t.Errorf("Bm(7) layout mismatch:\n got %q\nwant %q", out, want)
	}
	if len(list) != 2 {
		t.Errorf("Bm list size = %d, want 2", len(list))
	}
}

func TestBmZeroWidth(t *testing.T) {
	c := newFixedClock()
	out, _ := Bm(c, 0, nil, func(r *Report) []Tms {
		r.Run("z", func() {})
		return nil
	})
	const want = "       user     system      total        real\n" +
		"z  0.100000   0.050000   0.187500 (  1.000000)\n"
	if out != want {
		t.Errorf("Bm(0) layout mismatch:\n got %q\nwant %q", out, want)
	}
}

func TestBenchmarkWithExtraLines(t *testing.T) {
	c := newFixedClock()
	out, _ := Benchmark(c, CAPTION, 7, FORMAT, []string{">total:", ">avg:"},
		func(r *Report) []Tms {
			tf := r.Run("for:", func() {})
			tt := r.Run("times:", func() {})
			sum := tf.Add(tt)
			return []Tms{sum, sum.DivScalar(2)}
		})
	const want = "              user     system      total        real\n" +
		"for:      0.100000   0.050000   0.187500 (  1.000000)\n" +
		"times:    0.100000   0.050000   0.187500 (  1.000000)\n" +
		">total:   0.200000   0.100000   0.375000 (  2.000000)\n" +
		">avg:     0.100000   0.050000   0.187500 (  1.000000)\n"
	if out != want {
		t.Errorf("Benchmark extra-line layout mismatch:\n got %q\nwant %q", out, want)
	}
}

func TestBenchmarkNoCaptionDefaultFormatAndTmsLabel(t *testing.T) {
	c := newFixedClock()
	// Empty caption: no heading line. Empty format: FORMAT default. An extra Tms
	// with no override label uses its own label.
	out, _ := Benchmark(c, "", 0, "", nil, func(r *Report) []Tms {
		r.Run("a", func() {})
		return []Tms{NewTms(0, 0, 0, 0, 0, "summary")}
	})
	const want = "a  0.100000   0.050000   0.187500 (  1.000000)\n" +
		"summary  0.000000   0.000000   0.000000 (  0.000000)\n"
	if out != want {
		t.Errorf("Benchmark no-caption mismatch:\n got %q\nwant %q", out, want)
	}
}

func TestBmbmLayout(t *testing.T) {
	c := newFixedClock()
	job := NewJob(0).Item("sort!", func() {}).Item("sort", func() {})
	out, take := Bmbm(c, job)
	const want = "Rehearsal -----------------------------------------\n" +
		"sort!   0.100000   0.050000   0.187500 (  1.000000)\n" +
		"sort    0.100000   0.050000   0.187500 (  1.000000)\n" +
		"-------------------------------- total: 0.375000sec\n\n" +
		"            user     system      total        real\n" +
		"sort!   0.100000   0.050000   0.187500 (  1.000000)\n" +
		"sort    0.100000   0.050000   0.187500 (  1.000000)\n"
	if out != want {
		t.Errorf("Bmbm layout mismatch:\n got %q\nwant %q", out, want)
	}
	if len(take) != 2 || take[0].Label() != "sort!" || take[1].Label() != "sort" {
		t.Errorf("Bmbm take labels wrong: %v", take)
	}
}

func TestReportAccessors(t *testing.T) {
	c := newFixedClock()
	r := NewReport(c, 4, "%u")
	if r.Width() != 4 || r.Format() != "%u" {
		t.Errorf("report accessors: width=%d format=%q", r.Width(), r.Format())
	}
	r.Run("a", func() {})
	if len(r.List()) != 1 {
		t.Errorf("List len = %d, want 1", len(r.List()))
	}
	// A longer label widens the offset.
	r.Run("longlabel", func() {})
	if r.Width() != len("longlabel") {
		t.Errorf("width after long label = %d, want %d", r.Width(), len("longlabel"))
	}
}

func TestReportCaptionLineExtraLine(t *testing.T) {
	c := newFixedClock()
	r := NewReport(c, 8, FORMAT)
	if got := r.Caption(); got != "        "+CAPTION {
		t.Errorf("Caption() = %q", got)
	}
	tm := NewTms(1, 0, 0, 0, 0, "lbl")
	const wantLine = "lbl       1.000000   0.000000   1.000000 (  0.000000)\n"
	if got := r.Line(tm); got != wantLine {
		t.Errorf("Line() = %q, want %q", got, wantLine)
	}
	// ExtraLine with an override label.
	if got := r.ExtraLine(8, ">x:", tm); got != ">x:     "+tm.Format(FORMAT) {
		t.Errorf("ExtraLine override = %q", got)
	}
	// ExtraLine falling back to the Tms label.
	if got := r.ExtraLine(8, "", tm); got != "lbl     "+tm.Format(FORMAT) {
		t.Errorf("ExtraLine fallback = %q", got)
	}
}

func TestJobWidth(t *testing.T) {
	j := NewJob(3)
	if j.Width() != 3 {
		t.Errorf("initial width = %d, want 3", j.Width())
	}
	j.Item("xy", func() {}) // shorter than 3, width stays
	if j.Width() != 3 {
		t.Errorf("width after short = %d, want 3", j.Width())
	}
	j.Item("longer", func() {})
	if j.Width() != len("longer") {
		t.Errorf("width after long = %d, want %d", j.Width(), len("longer"))
	}
}

func TestStandaloneLayoutHelpers(t *testing.T) {
	if got := RehearsalHeader(7); got != "Rehearsal "+repeat("-", 7+len(CAPTION)-len("Rehearsal ")) {
		t.Errorf("RehearsalHeader = %q", got)
	}
	total := NewTms(0.1, 0.05, 0, 0, 1.0, "")
	const wantFooter = "--------------------------------- total: 0.150000sec\n\n"
	if got := RehearsalFooter(7, total); got != wantFooter {
		t.Errorf("RehearsalFooter = %q, want %q", got, wantFooter)
	}
	if got := TakeCaption(3); got != "   "+CAPTION {
		t.Errorf("TakeCaption = %q", got)
	}
}

func TestLjustEdgeCases(t *testing.T) {
	// String already at/over width is returned unchanged (covers the early
	// return in ljust, ljustFill, rjustFill).
	if got := ljust("abcdef", 3); got != "abcdef" {
		t.Errorf("ljust over width = %q", got)
	}
	if got := ljustFill("abcdef", 3, '-'); got != "abcdef" {
		t.Errorf("ljustFill over width = %q", got)
	}
	if got := rjustFill("abcdef", 3, '-'); got != "abcdef" {
		t.Errorf("rjustFill over width = %q", got)
	}
	if got := rjustFill("x", 4, '*'); got != "***x" {
		t.Errorf("rjustFill = %q, want ***x", got)
	}
}

// repeat is a tiny strings.Repeat shim kept local so the test reads clearly.
func repeat(s string, n int) string {
	out := ""
	for range n {
		out += s
	}
	return out
}

// Guard against accidental signature drift on ToH-free value model.
var _ = reflect.DeepEqual
