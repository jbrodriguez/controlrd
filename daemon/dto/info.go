package dto

// Info -
// No version || version 0
// Wake    Wake  `json:"wake"`
// Prefs   Prefs `json:"prefs"`
//
// Version 1
// Previous +
// Version int
// Ups []Sample
//
// Version 2
// + Available
type Info struct {
	Prefs    Prefs           `json:"prefs"`
	Features map[string]bool `json:"features"`
	Wake     Wake            `json:"wake"`
	Samples  []Sample        `json:"samples"`
	Version  int             `json:"version"`
}
