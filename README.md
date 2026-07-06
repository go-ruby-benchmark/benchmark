<p align="center"><img src="https://raw.githubusercontent.com/go-ruby-benchmark/brand/main/social/go-ruby-benchmark-benchmark.png" alt="go-ruby-benchmark/benchmark" width="720"></p>

# benchmark — go-ruby-benchmark

[![Docs](https://img.shields.io/badge/docs-mkdocs--material-DC2626)](https://go-ruby-benchmark.github.io/docs/)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.26.4%2B-00ADD8)](https://go.dev/dl/)
[![Coverage](https://img.shields.io/badge/coverage-100%25-1a7f37)](#tests--coverage)

**A pure-Go (no cgo) reimplementation of Ruby's standard-library
[Benchmark](https://docs.ruby-lang.org/en/master/Benchmark.html) module** — the
deterministic, interpreter-independent core of MRI 4.0.5 (`benchmark` 0.5.0): the
`Tms` measurement value type, its memberwise arithmetic and `%`-directive
formatting, and the report layout (`CAPTION`, label justification, the `bm` /
`bmbm` / `benchmark` tables). It reproduces MRI's formatted output **byte-for-byte**
— **without any Ruby runtime**.

It is the Benchmark backend for
[go-embedded-ruby](https://github.com/go-embedded-ruby/ruby), but is a
**standalone, reusable** module with no dependency on the Ruby runtime — a sibling
of [go-ruby-regexp](https://github.com/go-ruby-regexp/regexp) (the Onigmo engine),
[go-ruby-erb](https://github.com/go-ruby-erb/erb) (the ERB compiler), and
[go-ruby-yaml](https://github.com/go-ruby-yaml/yaml) (the Psych emitter/loader).

> **What it is — and isn't.** Formatting a `Tms`, doing its memberwise
> arithmetic, and laying out the report tables is fully deterministic and needs
> **no interpreter**, so it lives here as pure Go. The one impure ingredient — the
> **clock** — is injected: MRI's `Benchmark.measure` reads `Process.times` (the
> four CPU times) and `Process.clock_gettime(CLOCK_MONOTONIC)` (real time), and
> this library takes those readings through a small `Clock` interface. The host
> (go-embedded-ruby) wires in the real process clock; tests feed a fixed clock so
> every formatted line is reproducible.

## Features

Faithful port of `Benchmark` / `Benchmark::Tms`, validated against the `ruby`
binary on every supported platform:

- **`Tms` value type** — `utime`, `stime`, `cutime`, `cstime`, `real`, the derived
  `total = utime+stime+cutime+cstime`, and a `label`; with `ToA`, `ToS`, and the
  MRI-exact constructor semantics (a sum's label clears to `""`).
- **Memberwise arithmetic** — `Add` / `Sub` / `Mul` / `Div` against another `Tms`,
  and `AddScalar` / `SubScalar` / `MulScalar` / `DivScalar` against a number (the
  scalar is applied to **all five** fields, real included), exactly as MRI's
  `Tms#+ - * /` do.
- **`Format`** — the Benchmark `%`-extensions on top of standard printf
  directives: `%u` user, `%y` system, `%U`/`%Y` children's user/system, `%t`
  total, `%r` real (wrapped in parentheses), `%n` label — each preserving its
  flag/width/precision run (e.g. `%10.6u`). `%%` collapses and surplus args are
  dropped, matching Ruby's `String#%`. The default `FORMAT` is
  `"%10.6u %10.6y %10.6t %10.6r\n"`.
- **Report layout** — `CAPTION` (`"      user     system      total        real\n"`),
  label left-justification, and the full `bm(label_width)`, `bmbm` (rehearsal
  banner + total footer + take table), and general `benchmark(caption, width,
  format, *labels)` tables with trailing summary lines — all as pure functions
  over `Tms` values.
- **Injected clock** — `MeasureWith(clock, label, fn)`, `RealtimeWith`, `MsWith`,
  and `Tms.AddWith` take a `Clock` seam; the host supplies the real process clock,
  tests a deterministic one.

CGO-free, dependency-free, **100% test coverage**, `gofmt` + `go vet` clean, and
green across the six 64-bit Go targets (amd64, arm64, riscv64, loong64, ppc64le,
s390x) and three operating systems (Linux, macOS, Windows).

## Install

```sh
go get github.com/go-ruby-benchmark/benchmark
```

## Usage

```go
package main

import (
	"fmt"
	"time"

	benchmark "github.com/go-ruby-benchmark/benchmark"
)

// realClock implements benchmark.Clock with the process's real wall clock. A
// real host also fills in Process.times-style CPU numbers; here we only populate
// the monotonic real clock (as MRI does when CPU accounting is unavailable).
type realClock struct{ start time.Time }

func (c realClock) Times() (u, s, cu, cs float64) { return 0, 0, 0, 0 }
func (c realClock) Monotonic() float64            { return time.Since(c.start).Seconds() }

func main() {
	clock := realClock{start: time.Now()}

	// Benchmark.measure { ... } → a Tms.
	t := benchmark.MeasureWith(clock, "build", func() {
		_ = make([]byte, 1<<20)
	})
	fmt.Print(t.ToS())
	// e.g. "  0.000000   0.000000   0.000000 (  0.000123)"

	// Benchmark.bm(7) { |x| x.report("...") { ... } } → the table.
	out, _ := benchmark.Bm(clock, 7, nil, func(r *benchmark.Report) []benchmark.Tms {
		r.Run("warm:", func() { _ = make([]byte, 1<<20) })
		r.Run("cold:", func() { _ = make([]byte, 1<<20) })
		return nil
	})
	fmt.Print(out)
	//               user     system      total        real
	// warm:     0.000000   0.000000   0.000000 (  0.000xxx)
	// cold:     0.000000   0.000000   0.000000 (  0.000xxx)
}
```

## API

```go
// Constants matching Benchmark::CAPTION / FORMAT / BENCHMARK_VERSION.
const CAPTION = "      user     system      total        real\n"
const FORMAT  = "%10.6u %10.6y %10.6t %10.6r\n"
const BenchmarkVersion = "2002-04-25"

// Tms — the measurement value type (Benchmark::Tms).
func NewTms(utime, stime, cutime, cstime, real float64, label string) Tms
func (t Tms) Utime() float64
func (t Tms) Stime() float64
func (t Tms) Cutime() float64
func (t Tms) Cstime() float64
func (t Tms) Real() float64
func (t Tms) Total() float64   // utime+stime+cutime+cstime
func (t Tms) Label() string
func (t Tms) ToA() []any       // [label, utime, stime, cutime, cstime, real]
func (t Tms) ToS() string
func (t Tms) Format(format string, args ...any) string

func (t Tms) Add(o Tms) Tms        // Tms#+
func (t Tms) Sub(o Tms) Tms        // Tms#-
func (t Tms) Mul(o Tms) Tms        // Tms#*
func (t Tms) Div(o Tms) Tms        // Tms#/
func (t Tms) AddScalar(x float64) Tms
func (t Tms) SubScalar(x float64) Tms
func (t Tms) MulScalar(x float64) Tms
func (t Tms) DivScalar(x float64) Tms

// The injected clock seam (Process.times + clock_gettime(CLOCK_MONOTONIC)).
type Clock interface {
	Times() (utime, stime, cutime, cstime float64)
	Monotonic() float64
}
func MeasureWith(clock Clock, label string, fn func()) Tms   // Benchmark.measure
func RealtimeWith(clock Clock, fn func()) float64            // Benchmark.realtime
func MsWith(clock Clock, fn func()) float64                  // Benchmark.ms
func (t Tms) AddWith(clock Clock, fn func()) Tms             // Tms#add

// Report layout (Benchmark::Report / #benchmark / #bm / #bmbm).
type Report struct{ /* … */ }
func NewReport(clock Clock, width int, format string) *Report
func (r *Report) Run(label string, fn func()) Tms
func (r *Report) Line(t Tms) string
func (r *Report) Caption() string
func (r *Report) ExtraLine(labelWidth int, label string, t Tms) string

type Job struct{ /* … */ }
func NewJob(width int) *Job
func (j *Job) Item(label string, fn func()) *Job

func Benchmark(clock Clock, caption string, labelWidth int, format string,
	extraLabels []string, build func(*Report) []Tms) (string, []Tms)
func Bm(clock Clock, labelWidth int, extraLabels []string,
	build func(*Report) []Tms) (string, []Tms)
func Bmbm(clock Clock, job *Job) (string, []Tms)
```

## Tests & coverage

The suite pairs deterministic, ruby-free tests (which alone hold coverage at
100%, so the qemu cross-arch and Windows lanes pass the gate) with a
**differential MRI oracle**: the same fixed numbers and a patched, deterministic
clock are fed to the system `ruby`'s `Benchmark`, and the formatted output is
compared byte-for-byte. The oracle scripts `$stdout.binmode` so Windows text-mode
never pollutes the bytes, and skip themselves where `ruby` is absent or on
Windows.

```sh
COVERPKG=$(go list ./... | paste -sd, -)
go test -race -coverpkg="$COVERPKG" -coverprofile=cover.out ./...
go tool cover -func=cover.out | tail -1   # 100.0%
```

## License

BSD-3-Clause — see [LICENSE](LICENSE). Copyright the go-ruby-benchmark/benchmark authors.

## WebAssembly

Being pure Go (CGO=0), this library also compiles to **WebAssembly** — both
`GOOS=js GOARCH=wasm` (browser / Node.js) and `GOOS=wasip1 GOARCH=wasm` (WASI).
CI builds both targets on every push, alongside the six 64-bit native/qemu arches.

```sh
GOOS=js     GOARCH=wasm go build ./...   # browser / Node
GOOS=wasip1 GOARCH=wasm go build ./...   # WASI (wasmtime, wasmer, wasmedge, …)
```
