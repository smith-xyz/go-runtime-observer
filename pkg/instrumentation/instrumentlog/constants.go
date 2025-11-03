package instrumentlog

// env vars for instrument log and correlation cache
const (
	LOG_ENV_VAR             = "INSTRUMENTATION_LOG_PATH"
	ENV_MAX_SEEN_ENTRIES    = "INSTRUMENTATION_MAX_SEEN_ENTRIES"
	ENV_MAX_CORRELATIONS    = "INSTRUMENTATION_MAX_CORRELATIONS"
	ENV_CORRELATION_MAX_AGE = "INSTRUMENTATION_CORRELATION_MAX_AGE"
	ENV_CLEANUP_INTERVAL    = "INSTRUMENTATION_CLEANUP_INTERVAL"
	ENV_DEBUG_CORRELATION   = "INSTRUMENTATION_DEBUG_CORRELATION"
)

// defaults for correlation cache
const (
	defaultMaxSeenEntries    = 500000
	defaultMaxCorrelations   = 100000
	defaultCorrelationMaxAge = 50000
	defaultCleanupInterval   = 10000
)

// logger configuration
const (
	callerSkipDepth        = 3 // IMPORTANT: Call stack depth when runtime.Caller(3) executes: Frame 0=runtime.Caller, Frame 1=Log(), Frame 2=LogCall(), Frame 3=caller. The instrumented stdlib functions (reflect.ValueOf, etc.) call LogCall() directly, so we skip 3 frames to capture the actual application code caller, not the instrumentation wrapper.
	instrumentationPattern = "runtime_observe_instrumentation"
	defaultCallerName      = "unknown"
	logBufferInitialSize   = 256
)
