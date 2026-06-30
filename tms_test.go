package benchmark

import (
	"reflect"
	"testing"
)

// fixedClock is a deterministic Clock for the ruby-free tests: each Times call
// advances the cumulative user time by uStep and reports the children/system
// fields as fixed fractions of it; each Monotonic call advances by rStep. This
// makes every formatted line reproducible byte-for-byte.
type fixedClock struct {
	u, t         float64
	uStep, rStep float64
}

func newFixedClock() *fixedClock { return &fixedClock{uStep: 0.1, rStep: 1.0} }

func (c *fixedClock) Times() (float64, float64, float64, float64) {
	c.u += c.uStep
	return c.u, c.u / 2, c.u / 4, c.u / 8
}

func (c *fixedClock) Monotonic() float64 {
	c.t += c.rStep
	return c.t
}

func TestConstants(t *testing.T) {
	if CAPTION != "      user     system      total        real\n" {
		t.Errorf("CAPTION mismatch: %q", CAPTION)
	}
	if FORMAT != "%10.6u %10.6y %10.6t %10.6r\n" {
		t.Errorf("FORMAT mismatch: %q", FORMAT)
	}
	if BenchmarkVersion != "2002-04-25" {
		t.Errorf("BenchmarkVersion mismatch: %q", BenchmarkVersion)
	}
}

func TestTmsAccessors(t *testing.T) {
	tm := NewTms(1.0, 2.0, 0.5, 0.25, 3.5, "lbl")
	if tm.Utime() != 1.0 || tm.Stime() != 2.0 || tm.Cutime() != 0.5 ||
		tm.Cstime() != 0.25 || tm.Real() != 3.5 || tm.Label() != "lbl" {
		t.Errorf("accessor mismatch: %+v", tm.ToA())
	}
	// total = utime+stime+cutime+cstime (MRI: 3.75), NOT including real.
	if tm.Total() != 3.75 {
		t.Errorf("Total() = %v, want 3.75", tm.Total())
	}
}

func TestTmsToA(t *testing.T) {
	tm := NewTms(1.0, 2.0, 0.5, 0.25, 3.5, "lbl")
	got := tm.ToA()
	want := []any{"lbl", 1.0, 2.0, 0.5, 0.25, 3.5}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ToA() = %v, want %v", got, want)
	}
}

func TestTmsToSAndDefaultFormat(t *testing.T) {
	tm := NewTms(1.0, 2.0, 0.5, 0.25, 3.5, "lbl")
	const want = "  1.000000   2.000000   3.750000 (  3.500000)\n"
	if got := tm.ToS(); got != want {
		t.Errorf("ToS() = %q, want %q", got, want)
	}
	if got := tm.Format(""); got != want {
		t.Errorf("Format(\"\") = %q, want %q", got, want)
	}
}

func TestTmsFormatExtensions(t *testing.T) {
	tm := NewTms(1.0, 2.0, 0.5, 0.25, 3.5, "lbl")
	cases := []struct {
		fmt  string
		args []any
		want string
	}{
		// All extension letters plus a literal %%.
		{"%n %u %y %t %r %%", nil, "lbl 1.000000 2.000000 3.750000 (3.500000) %"},
		// A trailing standard directive consumes the extra arg.
		{"%u %5.2f", []any{99.0}, "1.000000 99.00"},
		// %% collapses even with no extra args.
		{"%u done %%", nil, "1.000000 done %"},
		// Width/justify on the label and a signed/precise utime.
		{"%-10n|", nil, "lbl       |"},
		{"%+.2u", nil, "+1.00"},
		// Children's user/system times.
		{"%U %Y", nil, "0.500000 0.250000"},
	}
	for _, c := range cases {
		if got := tm.Format(c.fmt, c.args...); got != c.want {
			t.Errorf("Format(%q, %v) = %q, want %q", c.fmt, c.args, got, c.want)
		}
	}
}

func TestTmsFormatExtraArgsIgnoredWithoutDirective(t *testing.T) {
	tm := NewTms(1.0, 2.0, 0.5, 0.25, 3.5, "x")
	// A format whose only directives are the (consumed) extensions leaves no
	// standard directive, so surplus args are dropped (Ruby String#% behaviour),
	// with no "%!(EXTRA …)" noise.
	if got := tm.Format("%u", 99.0); got != "1.000000" {
		t.Errorf("Format with surplus arg = %q, want %q", got, "1.000000")
	}
}

func TestTmsArithmeticTms(t *testing.T) {
	a := NewTms(2.0, 4.0, 1.0, 0.5, 8.0, "a")
	b := NewTms(1.0, 1.0, 0.5, 0.25, 2.0, "b")
	cases := []struct {
		name string
		got  []any
		want []any
	}{
		{"Add", a.Add(b).ToA(), []any{"", 3.0, 5.0, 1.5, 0.75, 10.0}},
		{"Sub", a.Sub(b).ToA(), []any{"", 1.0, 3.0, 0.5, 0.25, 6.0}},
		{"Mul", a.Mul(b).ToA(), []any{"", 2.0, 4.0, 0.5, 0.125, 16.0}},
		{"Div", a.Div(b).ToA(), []any{"", 2.0, 4.0, 2.0, 2.0, 4.0}},
	}
	for _, c := range cases {
		if !reflect.DeepEqual(c.got, c.want) {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
	// total of a product is recomputed from the new fields, not multiplied.
	if got := a.Mul(b).Total(); got != 6.625 {
		t.Errorf("Mul total = %v, want 6.625", got)
	}
}

func TestTmsArithmeticScalar(t *testing.T) {
	a := NewTms(2.0, 4.0, 1.0, 0.5, 8.0, "a")
	cases := []struct {
		name string
		got  []any
		want []any
	}{
		{"AddScalar", a.AddScalar(1.0).ToA(), []any{"", 3.0, 5.0, 2.0, 1.5, 9.0}},
		{"SubScalar", a.SubScalar(1.0).ToA(), []any{"", 1.0, 3.0, 0.0, -0.5, 7.0}},
		{"MulScalar", a.MulScalar(2.0).ToA(), []any{"", 4.0, 8.0, 2.0, 1.0, 16.0}},
		{"DivScalar", a.DivScalar(2.0).ToA(), []any{"", 1.0, 2.0, 0.5, 0.25, 4.0}},
	}
	for _, c := range cases {
		if !reflect.DeepEqual(c.got, c.want) {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
}

func TestMeasureWith(t *testing.T) {
	c := newFixedClock()
	tm := MeasureWith(c, "job", func() {})
	// One Times delta of uStep across the call: utime 0.1, stime 0.05, …; real 1.0.
	want := []any{"job", 0.1, 0.05, 0.025, 0.0125, 1.0}
	if !reflect.DeepEqual(tm.ToA(), want) {
		t.Errorf("MeasureWith = %v, want %v", tm.ToA(), want)
	}
}

func TestMeasureRunsBlock(t *testing.T) {
	c := newFixedClock()
	ran := false
	MeasureWith(c, "", func() { ran = true })
	if !ran {
		t.Error("MeasureWith did not run the block")
	}
}

func TestRealtimeWith(t *testing.T) {
	c := newFixedClock()
	if got := RealtimeWith(c, func() {}); got != 1.0 {
		t.Errorf("RealtimeWith = %v, want 1.0", got)
	}
}

func TestMsWith(t *testing.T) {
	c := newFixedClock()
	if got := MsWith(c, func() {}); got != 1000.0 {
		t.Errorf("MsWith = %v, want 1000.0", got)
	}
}

func TestTmsAddWith(t *testing.T) {
	c := newFixedClock()
	base := NewTms(1.0, 1.0, 1.0, 1.0, 1.0, "base")
	got := base.AddWith(c, func() {})
	// base + measured(0.1,0.05,0.025,0.0125,1.0); label cleared by memberwise.
	want := []any{"", 1.1, 1.05, 1.025, 1.0125, 2.0}
	if !reflect.DeepEqual(got.ToA(), want) {
		t.Errorf("AddWith = %v, want %v", got.ToA(), want)
	}
}
