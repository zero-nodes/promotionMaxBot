package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type User struct {
	Id int
	Id_max int
	Name string
	// user statuses: 
	//  -1 - user not found
	//	0 - free user
	//  1 - Waiting name from a user 
	//  2 - Waiting photo tiket from a user 
	Status int
	ShowAds bool
	Creates_at time.Time
}

func (s *PgStorage) AddUser(ctx context.Context, idMax int64, name string, status int) error {
    _, err := s.db.Exec(ctx,
        `INSERT INTO users (id_max, name, status)
         VALUES ($1, $2, $3)
         ON CONFLICT (id_max) DO UPDATE 
         SET name = EXCLUDED.name,
             status = EXCLUDED.status`,
        idMax, name, status,
    )
    if err != nil {
        return fmt.Errorf("AddUser failed: %w", err)
    }
    return nil
}

func (s *PgStorage) IsUserExistsById(ctx context.Context, idMax int64) (bool, error) {
    var exists bool
    err := s.db.QueryRow(ctx,
        "SELECT EXISTS(SELECT 1 FROM users WHERE id_max = $1)", idMax).
        Scan(&exists)

	if err != nil {
        return false, fmt.Errorf("Get exist user by id failed: %w", err)
	}

    return exists, err
}

func (s *PgStorage) GetUserShowAdsById(ctx context.Context, idMax int64) (bool, error) {
    var userShowAds bool
    err := s.db.QueryRow(ctx,
        "SELECT show_ads FROM users WHERE id_max = $1", idMax).
        Scan(&userShowAds)
	
	if err != nil {
		return true, fmt.Errorf("GetUserShowAdsById failed: %w", err)
    }

    return userShowAds, err
}

func (s *PgStorage) SetUserShowAdsById(ctx context.Context, idMax int64, newShowAds bool) error {
    _, err := s.db.Exec(ctx,
        `UPDATE users 
		 SET show_ads = $1
		 WHERE id_max = $2`,
        newShowAds, idMax,
    )
    if err != nil {
        return fmt.Errorf("Update User show_ads failed: %w", err)
    }
    return nil
}

func (s *PgStorage) GetUserStatusById(ctx context.Context, idMax int64) (int, error) {
    var userStatus int
    err := s.db.QueryRow(ctx,
        "SELECT status FROM users WHERE id_max = $1", idMax).
        Scan(&userStatus)
	
	if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return -1,  nil
		}
        
		return -1, fmt.Errorf("GetUserStatusById failed: %w", err)
    }

    return userStatus, err
}

func (s *PgStorage) SetUserStatusById(ctx context.Context, idMax int64, newStatus int) error {
    _, err := s.db.Exec(ctx,
        `UPDATE users 
		 SET status = $1
		 WHERE id_max = $2`,
        newStatus, idMax,
    )
    if err != nil {
        return fmt.Errorf("Update User failed: %w", err)
    }
    return nil
}

func (s *PgStorage) SetUserNameAndStatusById(ctx context.Context, idMax int64, newName string, newStatus int) error {
    _, err := s.db.Exec(ctx,
        `UPDATE users 
		 SET name = $1, status = $2
		 WHERE id_max = $3`,
        newName, newStatus, idMax,
    )
    if err != nil {
        return fmt.Errorf("Update User failed: %w", err)
    }
    return nil
}

func (s *PgStorage) GetUserNameById(ctx context.Context, idMax int64) (string, error) {
    var userName string
    err := s.db.QueryRow(ctx,
        "SELECT name FROM users WHERE id_max = $1", idMax).
        Scan(&userName)
	
	if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return "", err
		}
        
		return "", fmt.Errorf("GetUserNameById failed: %w", err)
    }

    return userName, err
}

func (s *PgStorage) GetListAllUserId(ctx context.Context) ([]int64, error) {
	rows, err := s.db.Query(ctx, "SELECT id_max FROM users")
	if err != nil {
		return nil, fmt.Errorf("error get list all user id -> %w", err)
	}
	defer rows.Close()
	
	var ids []int64
	for rows.Next() {
		var id int64

		err = rows.Scan(&id,)
		if err != nil {
			return nil, fmt.Errorf("error scan rows list user id -> %w", err)
		}

		ids = append(ids, id)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error rows iteration -> %w", err)
	}

	return ids, nil
}
