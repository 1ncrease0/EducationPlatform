package models

type Contents struct {
	Module  Module   `json:"module"`
	Lessons []Lesson `json:"lessons"`
}
