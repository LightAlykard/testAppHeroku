package pgstore

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/LightAlykard/testAppHeroku/app/repos/item"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v4/stdlib" // Postgresql driver
)

var _ item.ItemStore = &Items{}

type DBPgItem struct {
	ID          uuid.UUID `db:"id"`
	shortUrl    string    `db:"shortUrl"`
	longUrl     string    `db:"longUrl"`
	Count       int       `db:"count"`
	Permissions int       `db:"perms"`
}

type Items struct {
	db *sql.DB
}

func NewUsers(dsn string) *Items {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatal(err)
	}
	err = db.Ping()
	if err != nil {
		db.Close()
		log.Fatal(err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS public.items (
		id uuid NOT NULL,
		shortUrl varchar NOT NULL,
		longUrl varchar NULL,
		count int16 NULL,
		perms int2 NULL,
		CONSTRAINT items_pk PRIMARY KEY (id)
	)`)
	if err != nil {
		db.Close()
		log.Fatal(err)
	}

	us := &Items{
		db: db,
	}
	return us
}

func (us *Items) Close() {
	us.db.Close()
}

func (us *Items) Create(ctx context.Context, u item.Item) (*uuid.UUID, error) {
	uid := uuid.New()
	dbu := &DBPgItem{
		ID:          uid,
		shortUrl:    u.shortUrl,
		longUrl:     u.longUrl,
		Count:       u.Count,
		Permissions: u.Permissions,
	}

	_, err := us.db.ExecContext(ctx, `INSERT INTO items 
	(id, shortUrl, longUrl, count, perms)
	values ($1, $2, $3, $4, $5)`,
		dbu.ID,
		dbu.shortUrl,
		dbu.longUrl,
		dbu.Count,
		dbu.Permissions,
	)
	if err != nil {
		return nil, err
	}

	return &uid, nil
}

func (us *Items) Read(ctx context.Context, uid uuid.UUID) (*item.Item, error) {
	dbu := &DBPgItem{}
	rows, err := us.db.QueryContext(ctx, `SELECT id, shortUrl, longUrl, count, perms 
	FROM items WHERE id = $1`, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.Scan(
			&dbu.ID,
			&dbu.shortUrl,
			&dbu.longUrl,
			&dbu.Count,
			&dbu.Permissions,
		); err != nil {
			return nil, err
		}
	}

	return &item.Item{
		ID:          dbu.ID,
		shortUrl:    dbu.shortUrl,
		longUrl:     dbu.longUrl,
		Count:       dbu.Count,
		Permissions: dbu.Permissions,
	}, nil
}

func (us *Items) Delete(ctx context.Context, uid uuid.UUID) error {
	_, err := us.db.ExecContext(ctx, `UPDATE users SET deleted_at = $2 WHERE id = $1`,
		uid, time.Now(),
	)
	return err
}

func (us *Items) SearchItems(ctx context.Context, s string) (chan item.Item, error) {
	chout := make(chan item.Item, 100)

	go func() {
		defer close(chout)
		dbu := &DBPgItem{}

		rows, err := us.db.QueryContext(ctx, `
		SELECT id, shortUrl, longUrl, count, perms 
		FROM items WHERE shortUrl LIKE $1`, s+"%")
		if err != nil {
			log.Println(err)
			return
		}
		defer rows.Close()

		for rows.Next() {
			if err := rows.Scan(
				&dbu.ID,
				&dbu.shortUrl,
				&dbu.longUrl,
				&dbu.Count,
				&dbu.Permissions,
			); err != nil {
				log.Println(err)
				return
			}

			chout <- item.Item{
				ID:          dbu.ID,
				shortUrl:    dbu.shortUrl,
				longUrl:     dbu.longUrl,
				Count:       dbu.Count,
				Permissions: dbu.Permissions,
			}
		}
	}()

	return chout, nil
}
