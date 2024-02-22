package models

type AppConfiguration struct {
	TargetFile    string `json:"targetFile"`
	NumGoRoutines int    `json:"numGoRoutines"`
}

type URLFileConfiguration struct {
	URLs []string `json:"urls"`
}

type LogStructure struct {
	Logs []string `json:"logs"`
}
