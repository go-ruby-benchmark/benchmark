package benchmark

// Clock is the injected time source for MeasureWith. It abstracts the two clocks
// MRI's Benchmark.measure reads: Process.times (the four CPU times) and a
// monotonic real clock (Process.clock_gettime(CLOCK_MONOTONIC)).
//
// The host (go-embedded-ruby) supplies a Clock backed by the real process clock;
// tests supply a fixed or scripted Clock so the formatted output is fully
// deterministic.
type Clock interface {
	// Times returns the cumulative user, system, children's-user and
	// children's-system CPU seconds, like Process.times.
	Times() (utime, stime, cutime, cstime float64)
	// Monotonic returns a monotonically increasing real-time reading in
	// seconds, like Process.clock_gettime(Process::CLOCK_MONOTONIC).
	Monotonic() float64
}

// MeasureWith runs fn and returns a Tms holding the CPU and real time it took,
// labelled label. It mirrors Benchmark.measure: it samples the clock before and
// after fn and takes memberwise differences. The clock is injected via clock.
func MeasureWith(clock Clock, label string, fn func()) Tms {
	u0, s0, cu0, cs0 := clock.Times()
	r0 := clock.Monotonic()
	fn()
	u1, s1, cu1, cs1 := clock.Times()
	r1 := clock.Monotonic()
	return NewTms(u1-u0, s1-s0, cu1-cu0, cs1-cs0, r1-r0, label)
}

// RealtimeWith runs fn and returns the elapsed real time in seconds, mirroring
// Benchmark.realtime with an injected clock.
func RealtimeWith(clock Clock, fn func()) float64 {
	r0 := clock.Monotonic()
	fn()
	return clock.Monotonic() - r0
}

// MsWith runs fn and returns the elapsed real time in milliseconds, mirroring
// Benchmark.ms with an injected clock.
func MsWith(clock Clock, fn func()) float64 {
	return RealtimeWith(clock, fn) * 1000.0
}

// AddWith returns the memberwise sum of t and a fresh measurement of fn,
// mirroring Tms#add. The block is timed with the injected clock.
func (t Tms) AddWith(clock Clock, fn func()) Tms {
	return t.Add(MeasureWith(clock, "", fn))
}
