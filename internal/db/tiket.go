package db

import (
	"context"
	"fmt"
	"time"
)

type Ticket struct {
    ID        int
    Date      time.Time
    IDMax     int64
    Photo     []byte
    Status    int
	// ticket statuses:
	//  1 - new ticket (waiting for moderation)
	//  2 - declined (not counted)
	//  3 - approved (active coupon)
    CreatedAt time.Time
}

func (s *PgStorage) GetTiketsCount(ctx context.Context, idMax int64) (int, error) {
    var tiketsCount int
    err := s.db.QueryRow(ctx,
        "SELECT COUNT(*) FROM tiket WHERE id_max = $1 AND status = 3", idMax).
        Scan(&tiketsCount)
	
	if err != nil {
        return -1, fmt.Errorf("GetCountsTikets failed: %w", err)
    }

    return tiketsCount, err
}

func (s *PgStorage) AddTiket(ctx context.Context, idMax int64, photo []byte) error {
    today := time.Now().UTC().Format("2006-01-02")

    _, err := s.db.Exec(ctx,
        `INSERT INTO tiket (id_max, date, photo, status)
		VALUES ($1, $2::date, $3, $4)`,
        idMax, today, photo, 1,
    )

    if err != nil {
        return fmt.Errorf("Add tiket failed: %w", err)
    }
    return nil
}

func (s *PgStorage) GetTiketsCountToday(ctx context.Context, idMax int64) (int, error) {
    today := time.Now().UTC().Format("2006-01-02")

    var tiketsCount int
    err := s.db.QueryRow(ctx,
        `SELECT COUNT(*)
         FROM tiket
         WHERE id_max = $1 AND date = $2::date`,
        idMax, today).Scan(&tiketsCount)
    
    if err != nil {
        return -1, fmt.Errorf("GetCountsTiketsToday failed: %w", err)
    }

    return tiketsCount, nil
}
