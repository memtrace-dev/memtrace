package util

import (
	"crypto/rand"
	"time"

	"github.com/oklog/ulid/v2"
)

// GenerateID returns a new time-sortable unique ID.
func GenerateID() string {
	return ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String()
}
