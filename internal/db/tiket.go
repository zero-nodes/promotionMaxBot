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

func (s *PgStorage) setTicketStatusById(ctx context.Context, idTiket int, newStatus int) error {
    _, err := s.db.Exec(ctx,
        `UPDATE tiket
		 SET status = $1
		 WHERE id = $2`,
        newStatus, idTiket,
    )
    if err != nil {
        return fmt.Errorf("Update Ticket failed: %w", err)
    }
    return nil
}

func (s *PgStorage) ConfirmTiketById(ctx context.Context, idTiket int) error {
	err := s.setTicketStatusById(ctx, idTiket, 3)
	if err != nil {
        return fmt.Errorf("Confirm ticket failed: %w", err)
	}
	return nil
}

func (s *PgStorage) RejectTiketById(ctx context.Context, idTiket int) error {
	err := s.setTicketStatusById(ctx, idTiket, 2)
	if err != nil {
        return fmt.Errorf("Reject ticket failed: %w", err)
	}
	return nil
}

func (s *PgStorage) GetTicketsForModeration(ctx context.Context, limit int) ([]Ticket, error) {
	rows, err := s.db.Query(ctx,
		`SELECT 
			t.id,
			t.date,
			t.id_max,
			t.photo,
			t.status,
			t.created_at
		 FROM tiket t
		 WHERE t.status = 1
		 ORDER BY t.created_at ASC
		 LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("GetTicketsForModeration failed: %w", err)
	}
	defer rows.Close()

	var tickets []Ticket

	for rows.Next() {
		var t Ticket

		err := rows.Scan(&t.ID,	&t.Date, &t.IDMax, &t.Photo, &t.Status, &t.CreatedAt,)
		if err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		tickets = append(tickets, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration failed: %w", err)
	}

	return tickets, nil
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
