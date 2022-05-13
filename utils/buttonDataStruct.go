package utils

// used when a button needs to be updated in the context of interactions
type ButtonData struct {
	CustomID string
	Disabled bool
	Label    string
}

type ButtonDataManager struct {
	ButtonData []ButtonData
}
