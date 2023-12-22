package util

import (
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

func PGUUID() (pgtype.UUID, error) {
	uuid, err := uuid.NewRandom()
	if err != nil {
		return pgtype.UUID{Valid: false}, err
	}
	return pgtype.UUID{Bytes: uuid, Valid: true}, nil
}
