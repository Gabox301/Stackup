package detect

// ErlangDetector recognizes Erlang projects via rebar.config.
type ErlangDetector struct{}

// Name returns the ecosystem key.
func (ErlangDetector) Name() string { return "erlang" }

// Detect returns Erlang evidence: rebar.config alone grades Medium, plus
// rebar.lock or a .tool-versions erlang pin grades High with PackageManager
// rebar3. A .tool-versions erlang entry alone grades Low. Weak signals such
// as *.app.src, *.erl, or mix.exs never fire.
func (ErlangDetector) Detect(root string) []Evidence {
	if !exists(root, "rebar.config") {
		if v, ok := toolVersionsValue(root, "erlang"); ok {
			return []Evidence{{
				Ecosystem:   "erlang",
				Confidence:  ConfidenceLow,
				Signals:     []string{".tool-versions"},
				VersionHint: v,
			}}
		}
		return nil
	}

	ev := Evidence{
		Ecosystem:      "erlang",
		Confidence:     ConfidenceMedium,
		Signals:        []string{"rebar.config"},
		PackageManager: "rebar3",
	}
	if v, ok := toolVersionsValue(root, "erlang"); ok {
		ev.VersionHint = v
	}
	if exists(root, "rebar.lock") {
		ev.Signals = append(ev.Signals, "rebar.lock")
		ev.Confidence = ConfidenceHigh
	}
	if _, ok := toolVersionsValue(root, "erlang"); ok {
		if !containsSignal(ev.Signals, ".tool-versions") {
			ev.Signals = append(ev.Signals, ".tool-versions")
		}
		ev.Confidence = ConfidenceHigh
	}
	return []Evidence{ev}
}

// containsSignal reports whether signals already holds name.
func containsSignal(signals []string, name string) bool {
	for _, s := range signals {
		if s == name {
			return true
		}
	}
	return false
}
