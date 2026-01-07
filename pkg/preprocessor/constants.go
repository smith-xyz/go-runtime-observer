package preprocessor

const (
	InstrumentationMarker = "// INSTRUMENTED BY GO-RUNTIME-OBSERVER"

	InstrumentlogImportPath  = "runtime_observe_instrumentation/instrumentlog"
	FormatlogImportPath      = "runtime_observe_instrumentation/formatlog"
	InstrumentlogPackageName = "instrumentlog"
	FormatlogPackageName     = "formatlog"
	LogCallFunctionName      = "LogCall"

	InstrumentedSuffix = "_instrumented"
)
