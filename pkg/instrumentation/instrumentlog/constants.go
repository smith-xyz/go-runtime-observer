package instrumentlog

const (
	LOG_ENV_VAR          = "INSTRUMENTATION_LOG_PATH"
	ENV_MAX_SEEN_ENTRIES = "INSTRUMENTATION_MAX_SEEN_ENTRIES"
)

const (
	defaultMaxSeenEntries = 500000
)

const (
	callerSkipDepth        = 3
	instrumentationPattern = "runtime_observe_instrumentation"
	defaultCallerName      = "unknown"
	logBufferInitialSize   = 256
)
