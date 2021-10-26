package item

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type Item struct {
	ID          uuid.UUID
	ShortUrl    string
	LongUrl     string
	Count       int
	Permissions int
}

type ItemStore interface {
	Create(ctx context.Context, u Item) (*uuid.UUID, error)
	Read(ctx context.Context, uid uuid.UUID) (*Item, error)
	Delete(ctx context.Context, uid uuid.UUID) error
	SearchItems(ctx context.Context, s string) (chan Item, error)
}

type Items struct {
	ustore ItemStore
}

func NewUsers(ustore ItemStore) *Items {
	return &Items{
		ustore: ustore,
	}
}

func (us *Items) Create(ctx context.Context, u Item) (*Item, error) {
	id, err := us.ustore.Create(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("create item error: %w", err)
	}
	u.ID = *id
	return &u, nil
}

func (us *Items) Read(ctx context.Context, uid uuid.UUID) (*Item, error) {
	u, err := us.ustore.Read(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("read item error: %w", err)
	}
	return u, nil
}

func (us *Items) Delete(ctx context.Context, uid uuid.UUID) (*Item, error) {
	u, err := us.ustore.Read(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("search item error: %w", err)
	}
	return u, us.ustore.Delete(ctx, uid)
}

func (us *Items) SearchItems(ctx context.Context, s string) (chan Item, error) {
	// FIXME: здесь нужно использвоать паттерн Unit of Work
	// бизнес-транзакция
	chin, err := us.ustore.SearchItems(ctx, s)
	if err != nil {
		return nil, err
	}
	chout := make(chan Item, 100)
	go func() {
		defer close(chout)
		for {
			select {
			case <-ctx.Done():
				return
			case u, ok := <-chin:
				if !ok {
					return
				}
				u.Permissions = 0755
				chout <- u
			}
		}
	}()
	return chout, nil
}
