package model

type VMImage struct {
	Id          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	FilePath    string `json:"file_path"`
}
