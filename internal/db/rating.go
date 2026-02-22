package db

import (
	"context"
	"fmt"
)

type RatingRow struct {
	Num         int    `json:"num"`
	UserName    string `json:"name"`
	TiketsCount int    `json:"tickets"`
}


func (s *PgStorage) GetRating(ctx context.Context) ([]RatingRow, error) {
	rows, err := s.db.Query(ctx, 
	`SELECT
		 ROW_NUMBER() OVER (ORDER BY COUNT(t.id) DESC, MAX(t.date) DESC) AS num,
		 u.name,
		 COUNT(t.id) AS tickets
	 FROM users u
	 JOIN tiket t
		 ON u.id_max = t.id_max
		 AND t.status = 3
	 GROUP BY u.id, u.name, u.id_max
	 ORDER BY tickets DESC, MAX(t.date) DESC;`)

	if err != nil {
		return nil, fmt.Errorf("error get rating -> %w", err)
	}
	defer rows.Close()
	
	var rating []RatingRow
	for rows.Next() {
		var ratingRow RatingRow

		err = rows.Scan(&ratingRow.Num, &ratingRow.UserName, &ratingRow.TiketsCount)
		if err != nil {
			return nil, fmt.Errorf("error scan rows list rating -> %w", err)
		}

		rating = append(rating, ratingRow)
	}


	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error rows iteration -> %w", err)
	}

	return rating, nil
}
