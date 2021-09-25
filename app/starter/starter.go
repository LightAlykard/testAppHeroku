package starter

import (
	"context"
	"sync"

	"github.com/LightAlykard/testAppHeroku/app/repos/item"
)

type App struct {
	us *item.Items
}

func NewApp(ust item.ItemStore) *App {
	a := &App{
		us: item.NewUsers(ust),
	}
	return a
}

type HTTPServer interface {
	Start(us *item.Items)
	Stop()
}

func (a *App) Serve(ctx context.Context, wg *sync.WaitGroup, hs HTTPServer) {
	defer wg.Done()
	hs.Start(a.us)
	<-ctx.Done()
	hs.Stop()
}
