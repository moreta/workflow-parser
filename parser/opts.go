package parser

type OptionFunc func(*parseState)

func WithSuppressSeverity(sev Severity) OptionFunc {
	return func(ps *parseState) {
		ps.suppressSeverity = sev
	}
}
