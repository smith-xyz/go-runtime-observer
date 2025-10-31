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
