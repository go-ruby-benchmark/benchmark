package benchmark

import (
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

// rubyOracle runs `ruby -rbenchmark -e <script>` and returns its stdout. It
// skips the test when ruby is unavailable or on Windows (where the runner has no
// ruby and text-mode would corrupt the bytes anyway). The script must
// $stdout.binmode and emit raw bytes so the comparison is exact on every OS.
func rubyOracle(t *testing.T, script string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("ruby oracle skipped on Windows")
	}
	if _, err := exec.LookPath("ruby"); err != nil {
		t.Skip("ruby not installed; skipping MRI oracle")
	}
	full := "$stdout.binmode if $stdout.respond_to?(:binmode)\nrequire 'benchmark'\n" + script
	out, err := exec.Command("ruby", "-e", full).Output()
	if err != nil {
		t.Fatalf("ruby oracle failed: %v", err)
	}
	return string(out)
}

// TestOracleTmsFormat differentially checks Tms formatting/arithmetic against the
// system MRI: the same fixed numbers are fed to Benchmark::Tms in Ruby and to Tms
// here, and the printed strings must match byte-for-byte.
func TestOracleTmsFormat(t *testing.T) {
	tm := NewTms(1.0, 2.0, 0.5, 0.25, 3.5, "lbl")
	script := `t = Benchmark::Tms.new(1.0, 2.0, 0.5, 0.25, 3.5, "lbl")
print t.to_s
print t.format("%n %u %y %t %r %%")
print "\n"
print t.format("%U %Y")
print "\n"
a = Benchmark::Tms.new(2.0,4.0,1.0,0.5,8.0,"a")
b = Benchmark::Tms.new(1.0,1.0,0.5,0.25,2.0,"b")
print (a+b).format, (a-b).format, (a*b).format, (a/b).format
print (a*2).format, (a/2).format
`
	want := rubyOracle(t, script)
	var sb strings.Builder
	sb.WriteString(tm.ToS())
	sb.WriteString(tm.Format("%n %u %y %t %r %%"))
	sb.WriteString("\n")
	sb.WriteString(tm.Format("%U %Y"))
	sb.WriteString("\n")
	a := NewTms(2.0, 4.0, 1.0, 0.5, 8.0, "a")
	b := NewTms(1.0, 1.0, 0.5, 0.25, 2.0, "b")
	sb.WriteString(a.Add(b).Format(""))
	sb.WriteString(a.Sub(b).Format(""))
	sb.WriteString(a.Mul(b).Format(""))
	sb.WriteString(a.Div(b).Format(""))
	sb.WriteString(a.MulScalar(2).Format(""))
	sb.WriteString(a.DivScalar(2).Format(""))
	if got := sb.String(); got != want {
		t.Errorf("Tms oracle mismatch:\n got %q\nwant %q", got, want)
	}
}

// TestOracleConstants checks the caption/format/version constants against MRI.
func TestOracleConstants(t *testing.T) {
	want := rubyOracle(t, `print Benchmark::CAPTION, Benchmark::FORMAT, Benchmark::BENCHMARK_VERSION`)
	got := CAPTION + FORMAT + BenchmarkVersion
	if got != want {
		t.Errorf("constants oracle mismatch:\n got %q\nwant %q", got, want)
	}
}

// TestOracleBmLayout differentially checks the bm table layout. MRI's clock is
// patched to a deterministic stub mirroring the Go fixedClock (Process.times
// advances cumulative user time by 0.1; clock_gettime by 1.0), so both sides are
// reproducible and must match byte-for-byte.
func TestOracleBmLayout(t *testing.T) {
	script := `require 'stringio'
module Det
  @t=0.0; @u=0.0
  def self.times; @u+=0.1; Process::Tms.new(@u,@u/2,@u/4,@u/8); end
  def self.mono;  @t+=1.0; @t; end
end
module Process
  class << self
    def times; Det.times; end
    def clock_gettime(*); Det.mono; end
  end
end
def cap; old=$stdout; $stdout=StringIO.new; yield; $stdout.string; ensure; $stdout=old; end
print(cap{ Benchmark.bm(7) { |x| x.report("for:"){nil}; x.report("times:"){nil} } })
`
	want := rubyOracle(t, script)
	c := newFixedClock()
	got, _ := Bm(c, 7, nil, func(r *Report) []Tms {
		r.Run("for:", func() {})
		r.Run("times:", func() {})
		return nil
	})
	if got != want {
		t.Errorf("bm oracle mismatch:\n got %q\nwant %q", got, want)
	}
}

// TestOracleBmbmLayout differentially checks the full bmbm rehearsal+take report.
func TestOracleBmbmLayout(t *testing.T) {
	script := `require 'stringio'
module Det
  @t=0.0; @u=0.0
  def self.times; @u+=0.1; Process::Tms.new(@u,@u/2,@u/4,@u/8); end
  def self.mono;  @t+=1.0; @t; end
end
module Process
  class << self
    def times; Det.times; end
    def clock_gettime(*); Det.mono; end
  end
end
module GC; def self.start; end; end
def cap; old=$stdout; $stdout=StringIO.new; yield; $stdout.string; ensure; $stdout=old; end
print(cap{ Benchmark.bmbm { |x| x.report("sort!"){nil}; x.report("sort"){nil} } })
`
	want := rubyOracle(t, script)
	c := newFixedClock()
	job := NewJob(0).Item("sort!", func() {}).Item("sort", func() {})
	got, _ := Bmbm(c, job)
	if got != want {
		t.Errorf("bmbm oracle mismatch:\n got %q\nwant %q", got, want)
	}
}
