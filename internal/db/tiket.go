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


