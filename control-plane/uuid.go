package main

import (
	"crypto/rand"
	"fmt"

	"github.com/google/uuid"
)

func init() {
	uuid.SetRand(rand.Reader)
}

func newUUID() uuid.UUID {
	id, err := uuid.NewRandom()
	if err != nil {
		panic(fmt.Sprintf("failed to generate UUID: %v", err))
	}
	return id
}
