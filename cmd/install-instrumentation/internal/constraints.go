package internal

import "slices"

const (
	ALLOWED_INSTRUMENT_FUNCTION = "InstrumentPackageFiles"
	
	REQUIRED_INSTRUMENT_ARG_COUNT    = 2
	REQUIRED_INSTRUMENT_RESULT_COUNT = 2
	REQUIRED_REPARSE_RESULT_COUNT    = 2
)

var AllowedReparseFunctions = []string{
	"buildContext.ImportDir",
	"ctxt.ImportDir",
}

func isAllowedReparseFunction(fn string) bool {
	return slices.Contains(AllowedReparseFunctions, fn)
}

