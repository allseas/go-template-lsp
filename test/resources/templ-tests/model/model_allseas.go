package model

// Model is the top-level data passed to the template.
// gotype hint: {{- /*gotype: cg/model.Model*/ -}}
type Model struct {
	Machine        Instance
	Instances      []Instance
	Commands       []CommandGroup
	AlarmInstances []AlarmInstance
	ControlSystem  ControlSystem
	LoggerInfo     LoggerInfo
}

// Instance represents a controller block instance.
// gotype hint: {{- /*gotype: cg/model.Instance*/ -}}
type Instance struct {
	Block                Block
	IdentifyingSelection IdentifyingSelection
	Inputs               []SignalInstance
	Outputs              []SignalInstance
}

// Block describes a controller block.
type Block struct {
	Name        string
	IsComponent bool
	Logging     []LoggingField
}

// IdentifyingSelection holds an ordered set of identifying key-value pairs.
type IdentifyingSelection struct {
	Order []string
	vals  map[string]string
}

// Get returns the value for the given key.
func (s IdentifyingSelection) Get(key string) string {
	return s.vals[key]
}

// SignalInstance is one signal (input or output) on an instance.
type SignalInstance struct {
	Name   string
	Signal Signal
}

// Signal describes a data signal.
type Signal struct {
	Datatype Datatype
}

// Datatype holds type information for a signal.
type Datatype struct {
	Golang string
}

// CommandGroup holds all instances of a single command type.
type CommandGroup struct {
	CommandInstances []CommandInstance
}

// CommandInstance is a single use of a command.
type CommandInstance struct {
	Command Command
	Inputs  []SignalInstance
	Outputs []SignalInstance
}

// Command describes a hardware command.
type Command struct {
	Name               string
	HasHardwareInputs  bool
	HasHardwareOutputs bool
}

// AlarmInstance pairs an alarm definition with its instance.
type AlarmInstance struct {
	Instance Instance
	Alarm    Alarm
}

// Alarm describes an alarm.
type Alarm struct {
	Name string
}

// LoggingField is a field exposed through a block's logging list.
type LoggingField struct {
	Name     string
	Datatype Datatype
}

// ControlSystem is the top-level control system.
type ControlSystem struct {
	Block Block
}

// LoggerInfo holds logger configuration.
type LoggerInfo struct {
	EnablePing      bool
	UdpHeader       UdpHeader
	IgnoreCRCErrors bool
}

// UdpHeader holds UDP-level header configuration.
type UdpHeader struct {
	SystemInstanceID string
}

// Offset is the byte/bit position of a field in the binary layout.
type Offset struct {
	Byte int
	Bit  int
}

// Layout tracks the current position while iterating fields.
type Layout struct {
	bytePos int
	bitPos  int
}

// Add advances the layout by one field and returns the current offset.
func (l *Layout) Add(signal interface{}) Offset {
	o := Offset{Byte: l.bytePos}
	l.bytePos++
	return o
}

// AddBit advances by one bit and returns the current offset.
func (l *Layout) AddBit() Offset {
	o := Offset{Byte: l.bytePos, Bit: l.bitPos}
	l.bitPos++
	return o
}

// Align advances to the next byte boundary.
func (l *Layout) Align() string {
	if l.bitPos > 0 {
		l.bytePos++
		l.bitPos = 0
	}
	return ""
}

// StartBitSet marks the start of a bit-packed region at byte n.
func (l *Layout) StartBitSet(n int) string {
	l.bytePos = n
	l.bitPos = 0
	return ""
}

// EndBitSet finalises a bit-packed region.
func (l *Layout) EndBitSet() string {
	l.bytePos++
	l.bitPos = 0
	return ""
}
