package models

import "github.com/google/uuid"

const (
	ClientRole = "client"
	AdminRole  = "admin"
)

type User struct {
	ID       uuid.UUID
	Username string
	Password string
	Email    string
	Roles    []string
}
