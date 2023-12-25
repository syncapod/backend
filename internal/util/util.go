package util

import (
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Err is a wrapper to quickly log an error
func Err(err error) slog.Attr {
	return slog.Any("error", err)
}

func PGNow() pgtype.Timestamptz {
	return PGFromTime(time.Now())
}

func PGFromTime(time time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{
		Time:             time,
		InfinityModifier: pgtype.Finite,
		Valid:            true,
	}
}

func PGDateFromTime(time time.Time) pgtype.Date {
	return pgtype.Date{
		Time:             time,
		InfinityModifier: pgtype.Finite,
		Valid:            true,
	}
}

func PGNewUUID() (pgtype.UUID, error) {
	uuid, err := uuid.NewRandom()
	if err != nil {
		return pgtype.UUID{Valid: false}, err
	}
	return PGUUID(uuid), nil
}

func PGUUID(uuid uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: uuid, Valid: true}
}

func UUIDFromPG(id pgtype.UUID) (uuid.UUID, error) {
	if !id.Valid {
		return uuid.UUID{}, errors.New("error pgtype.UUID is not valid")
	}
	return uuid.FromBytes(id.Bytes[:])
}

func StringFromPGUUID(id pgtype.UUID) (string, error) {
	newID, err := UUIDFromPG(id)
	if err != nil {
		return "", err
	}
	return newID.String(), nil
}

func PGBool(value bool) pgtype.Bool {
	return pgtype.Bool{
		Bool:  value,
		Valid: true,
	}
}
