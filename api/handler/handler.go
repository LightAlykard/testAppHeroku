package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/LightAlykard/testAppHeroku/api/openapi"
	"github.com/LightAlykard/testAppHeroku/app/repos/item"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Router struct {
	chi.Router
	us *item.Items
}

func NewRouter(us *item.Items) *Router {
	r := chi.NewRouter()
	r.Use(AuthMiddleware)

	rt := &Router{
		Router: r,
		us:     us,
	}

	swg, err := openapi.GetSwagger()
	if err != nil {
		log.Fatal("swagger fail")
	}

	r.Mount("/", openapi.Handler(rt))

	// r.HandleFunc("/delete", r.AuthMiddleware(http.HandlerFunc(r.DeleteUser)).ServeHTTP)
	// r.HandleFunc("/search", r.AuthMiddleware(http.HandlerFunc(r.SearchUser)).ServeHTTP)

	rt.Get("/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		enc := json.NewEncoder(w)
		_ = enc.Encode(swg)
	})

	return rt
}

type Item struct {
	ID         uuid.UUID `json:"id"`
	shortUrl   string    `json:"shortUrl"`
	longUrl    string    `json:"longUrl"`
	Count      int       `json:"count"`
	Permission int       `json:"perms"`
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if u, p, ok := r.BasicAuth(); !ok || !(u == "admin" && p == "admin") {
				http.Error(w, "unautorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		},
	)
}

func (rt *Router) PostCreate(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	u := Item{}
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	bu := item.Item{
		shortUrl: u.shortUrl,
		longUrl:  u.longUrl,
		//надо ли сюда count
	}

	nbu, err := rt.us.Create(r.Context(), bu)
	if err != nil {
		log.Println(err)
		http.Error(w, "error when creating", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)

	_ = json.NewEncoder(w).Encode(
		Item{
			ID:         nbu.ID,
			shortUrl:   nbu.longUrl,
			longUrl:    nbu.longUrl,
			Count:      nbu.Count,
			Permission: nbu.Permissions,
		},
	)
}

// read/{uid}
func (rt *Router) GetReadId(w http.ResponseWriter, r *http.Request, suid string) {
	uid, err := uuid.Parse(suid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if (uid == uuid.UUID{}) {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	nbu, err := rt.us.Read(r.Context(), uid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
		} else {
			http.Error(w, "error when reading", http.StatusInternalServerError)
		}
		return
	}

	_ = json.NewEncoder(w).Encode(
		Item{
			ID:         nbu.ID,
			shortUrl:   nbu.longUrl,
			longUrl:    nbu.longUrl,
			Count:      nbu.Count,
			Permission: nbu.Permissions,
		},
	)
}

func (rt *Router) DeleteDeleteId(w http.ResponseWriter, r *http.Request, suid string) {
	uid, err := uuid.Parse(suid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if (uid == uuid.UUID{}) {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	nbu, err := rt.us.Delete(r.Context(), uid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
		} else {
			http.Error(w, "error when reading", http.StatusInternalServerError)
		}
		return
	}

	_ = json.NewEncoder(w).Encode(
		Item{
			ID:         nbu.ID,
			shortUrl:   nbu.longUrl,
			longUrl:    nbu.longUrl,
			Count:      nbu.Count,
			Permission: nbu.Permissions,
		},
	)
}

// /search?q=...
func (rt *Router) FindItems(w http.ResponseWriter, r *http.Request, q string) {
	ch, err := rt.us.SearchItems(r.Context(), q)
	if err != nil {
		http.Error(w, "error when reading", http.StatusInternalServerError)
		return
	}

	enc := json.NewEncoder(w)

	first := true
	fmt.Fprintf(w, "[")
	defer fmt.Fprintf(w, "]")

	for {
		select {
		case <-r.Context().Done():
			return
		case u, ok := <-ch:
			if !ok {
				return
			}
			if first {
				first = false
			} else {
				fmt.Fprintf(w, ",")
			}
			_ = enc.Encode(
				Item{
					ID:         u.ID,
					shortUrl:   u.longUrl,
					longUrl:    u.longUrl,
					Count:      u.Count,
					Permission: u.Permissions,
				},
			)
			w.(http.Flusher).Flush()
		}
	}
}
