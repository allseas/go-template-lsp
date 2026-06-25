package handlers

type renameTestCase struct {
	name       string
	src        string
	typeKind   string
	line       uint32
	char       uint32
	chars      []uint32
	newName    string
	wantNil    bool
	wantText   string
	wantCount  int
	wantLen    uint32
	wantStarts []uint32
}

var renameTestCases = []renameTestCase{
	{
		name: "variable all occurrences",
		src: `{{ $x := 1 }}
			{{ $x }}
			{{ $x }}`,
		line:      0,
		char:      4,
		newName:   "$y",
		wantText:  "$y",
		wantCount: 3,
	},
	{
		name:      "variable adds dollar prefix",
		src:       `{{ $x := 1 }}{{ $x }}`,
		line:      0,
		char:      4,
		newName:   "renamed",
		wantText:  "$renamed",
		wantCount: 2,
	},
	{
		name:      "variable leaves chain intact",
		src:       `{{ $x := . }}{{ $x.Name }}`,
		line:      0,
		char:      4,
		newName:   "$y",
		wantText:  "$y",
		wantCount: 2,
		wantLen:   2,
	},
	{
		name: "identifier",
		src: `{{ printf "a" }}
			{{ printf "b" }}`,
		line:      0,
		char:      4,
		newName:   "sprintf",
		wantText:  "sprintf",
		wantCount: 2,
	},
	{
		name:    "cursor on non node",
		src:     `hello {{ $x := 1 }}`,
		line:    0,
		char:    1,
		newName: "$y",
		wantNil: true,
	},
	{
		name:    "empty new name",
		src:     `{{ $x := 1 }}{{ $x }}`,
		line:    0,
		char:    4,
		newName: "   ",
		wantNil: true,
	},
	{
		name:      "field on dot",
		src:       `{{ .CustomerName }} {{ .CustomerName }}`,
		typeKind:  "order",
		line:      0,
		char:      5,
		newName:   "ClientName",
		wantText:  "ClientName",
		wantCount: 2,
	},
	{
		name:      "method on dot",
		src:       `{{ .DisplayName }} {{ .DisplayName }}`,
		typeKind:  "order",
		line:      0,
		char:      5,
		newName:   "FullName",
		wantText:  "FullName",
		wantCount: 2,
	},
	{
		name:      "nested field only edits target segment",
		src:       `{{ .Address.City }} {{ .Address.City }}`,
		typeKind:  "order",
		line:      0,
		char:      13,
		newName:   "Town",
		wantText:  "Town",
		wantCount: 2,
		wantLen:   4,
	},
	{
		name:      "nested field does not match sibling",
		src:       `{{ .Address.City }} {{ .Address.Street }}`,
		typeKind:  "order",
		line:      0,
		char:      13,
		newName:   "Town",
		wantText:  "Town",
		wantCount: 1,
	},
	{
		name:      "field across variable and dot",
		src:       `{{ $o := . }}{{ $o.CustomerName }}{{ .CustomerName }}`,
		typeKind:  "order",
		line:      0,
		char:      20,
		newName:   "ClientName",
		wantText:  "ClientName",
		wantCount: 2,
	},
	{
		name:      "field via variable leaves base untouched",
		src:       `{{ $o := . }}{{ $o.DisplayName }}`,
		typeKind:  "order",
		line:      0,
		char:      22,
		newName:   "FullName",
		wantText:  "FullName",
		wantCount: 1,
		wantLen:   11,
	},
	{
		name:     "field without type info",
		src:      `{{ .CustomerName }}`,
		typeKind: "source",
		line:     0,
		char:     5,
		newName:  "ClientName",
		wantNil:  true,
	},
	{
		name:      "middle chain segment",
		src:       `{{ .Address.Info.Desc }}{{ if .Address.Info.Desc.Info1 }}x{{ end }}`,
		typeKind:  "chain",
		line:      0,
		char:      13,
		newName:   "Renamed",
		wantText:  "Renamed",
		wantCount: 2,
		wantLen:   4,
	},
	{
		name:      "middle chain leaves similar leaf intact",
		src:       `{{ if .Address.Info.Desc.Info1 }}x{{ end }}`,
		typeKind:  "chain",
		line:      0,
		char:      16,
		newName:   "Renamed",
		wantText:  "Renamed",
		wantCount: 1,
		wantLen:   4,
	},
	{
		name:      "leaf chain segment",
		src:       `{{ if .Address.Info.Desc.Info1 }}x{{ end }}`,
		typeKind:  "chain",
		line:      0,
		char:      27,
		newName:   "Renamed",
		wantText:  "Renamed",
		wantCount: 1,
		wantLen:   5,
	},
	{
		name:       "middle chain no spaces word boundaries",
		src:        `{{.Address.Info.Desc}} ({{if .Address.Info.Desc.Info1}}x{{end}})`,
		typeKind:   "chain",
		line:       0,
		chars:      []uint32{11, 12, 13, 14, 15},
		newName:    "Renamed",
		wantText:   "Renamed",
		wantCount:  2,
		wantLen:    4,
		wantStarts: []uint32{11, 38},
	},
}
