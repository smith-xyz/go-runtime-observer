package config

type VersionConfig struct {
	Go          string
	BaseVersion string
	Notes       string
	Injections  []InjectionConfig
	Patches     []PatchConfig
	Overrides   map[string]VersionOverride
}

type VersionOverride struct {
	Injections []InjectionOverride
	Patches    []PatchConfig
}

type InjectionOverride struct {
	Name string
	Line int
}

type PatchConfig struct {
	Name        string
	TargetFile  string
	Description string
	Find        string
	Replace     string
}

type InjectionConfig struct {
	Name        string
	TargetFile  string
	Line        int
	Description string
	Instrument  InstrumentCall
	Reparse     ReparseCall
}

type InstrumentCall struct {
	Function string
	Args     []string
	Result   []string
}

type ReparseCall struct {
	Result   []string
	Function string
	Args     []string
}
