package types

type LoggerType string

const (
	LoggerTypeInstrument LoggerType = "instrumentlog"
	LoggerTypeFormat     LoggerType = "formatlog"
)

// StdlibMethodInstrumentation configures how to instrument stdlib methods
type StdlibMethodInstrumentation struct {
	// ReceiverType specifies the type name of the method receiver (e.g., "Value" for reflect.Value methods)
	ReceiverType string

	// MethodNames lists all method names to instrument with logging calls
	// Example: []string{"Call", "Set", "MethodByName"}
	MethodNames []string

	// CorrelationRecordingMethods lists methods that should record correlations for call graph reconstruction
	// These methods return values that are later used in calls (e.g., MethodByName returns a Value used in Call)
	// Example: []string{"MethodByName", "Method"}
	CorrelationRecordingMethods []string

	// MethodIdentifierExtractors maps method names to extraction strategies for correlation identifiers
	// Format: "param:name" extracts function parameter named "name"
	//         "call:0" extracts first argument from the return expression's call
	// Example: map[string]string{"MethodByName": "param:name", "Method": "call:0"}
	MethodIdentifierExtractors map[string]string

	// ReturnExpressionMethods maps correlation recording methods to which methods appear in their return statements
	// Empty slice means check for the same method name; populated slice checks for those methods
	// Example: map[string][]string{"MethodByName": {"Method"}} means MethodByName returns v.Method(...)
	//          map[string][]string{"Method": {}} means Method returns itself or a direct value
	ReturnExpressionMethods map[string][]string

	// CorrelationLookupMethods lists methods that should perform correlation lookup when called
	// These methods consume correlations recorded by CorrelationRecordingMethods
	// Example: []string{"Call", "CallSlice"} for reflect.Value methods that invoke dynamically looked-up methods
	CorrelationLookupMethods []string
}

// StdlibASTInstrumentation configures AST-level instrumentation for a stdlib package
type StdlibASTInstrumentation struct {
	PackageName string
	Functions   []string
	Methods     []StdlibMethodInstrumentation
	Logger      LoggerType
}

// InstrumentedPackage defines a wrapper package for instrumented stdlib functions
type InstrumentedPackage struct {
	Pkg       string   `json:"pkg"`
	Functions []string `json:"functions"`
}

// Registry interface for checking instrumentation configuration
// This allows AST operations to work without importing the concrete Registry type
type Registry interface {
	IsInstrumented(stdlibPackage, functionName string) bool
	GetInstrumentedImportPath(stdlibPackage string) (string, bool)
}
