package models

import (
	"time"

	"github.com/NikoMalik/uuid"
)

type SocketData struct {
	ID   *uuid.UUID
	Data []byte
}

type RandomData struct {
	ID          int
	Name        string
	Description string
	Value       int
	CreatedAt   time.Time
}
