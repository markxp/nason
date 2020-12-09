package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/mux"

	"database/sql"
)

// Article is an article
type Article struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Desc    string `json:"description"`
	Content string `json:"content"`
}

// ArticleService let you store articles.
// It's the central of our service. It contains all methods we can do with it, and may using external service or storage.
type ArticleService struct {
	DB *sql.DB
}

// Prepare setup DB schemas
func (s ArticleService) Prepare(ctx context.Context) {
	stat := `CREATE TABLE articles (id BIGSERIAL NOT NULL PRIMARY KEY, title TEXT, description TEXT, content TEXT);`
	if s.DB == nil {
		panic("no existing database")
	}
	if _, err := s.DB.ExecContext(ctx, stat); err != nil {
		panic(err)
	}
}

// Create creates a article
func (s ArticleService) Create(ctx context.Context, i Article) error {
	stat := `INSERT INTO articles (title, description, content) VALUES(?,?,?);`
	if s.DB == nil {
		panic("no existing database")
	}
	_, err := s.DB.ExecContext(ctx, stat, i.Title, i.Desc, i.Content)
	return err
}

// Get reads an article
func (s ArticleService) Get(ctx context.Context, id string) (*Article, error) {
	stat := `SELECT id, title, description, content FROM articles WHERE id = ?;`
	if s.DB == nil {
		panic("no existing database")
	}
	rows, err := s.DB.QueryContext(ctx, stat, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var article Article
	if rows.Next() {
		err := rows.Scan(&article.ID, &article.Title, &article.Desc, &article.Content)
		if err != nil {
			return nil, err
		}
	}
	return &article, err
}

// List reads all articles
func (s ArticleService) List(ctx context.Context) ([]Article, error) {
	stat := `SELECT id, title, description, content FROM articles;`
	if s.DB == nil {
		panic("no existing database")
	}
	rows, err := s.DB.QueryContext(ctx, stat)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ret := make([]Article, 0, 20)
	for rows.Next() {
		fmt.Println("got 1 record")
		var article Article
		err := rows.Scan(&article.ID, &article.Title, &article.Desc, &article.Content)
		if err != nil {
			log.Println(err)
			continue
		}
		ret = append(ret, article)
	}
	return ret, err

}

// Delete deletes an article
func (s ArticleService) Delete(ctx context.Context, id string) error {
	stat := `DELETE FROM article WHERE id = ?;`
	_, err := s.DB.ExecContext(ctx, stat, id)
	return err
}

var defaultHandler http.Handler

// RESTful returns RESTful API of article service.
// It contains its routes and handle http requests.
func (s ArticleService) RESTful() http.Handler {
	o := sync.Once{}
	o.Do(s.registerRoutes)

	return defaultHandler
}

func (s ArticleService) registerRoutes() {
	m := mux.NewRouter().StrictSlash(false)
	defaultHandler = m

	m.HandleFunc("/list", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		ctx := r.Context()
		articles, err := s.List(ctx)
		if err != nil {
			http.Error(w, "could not read data", http.StatusInternalServerError)
			return
		}

		b := &bytes.Buffer{}
		if err := json.NewEncoder(b).Encode(articles); err != nil {
			http.Error(w, "could not encode json", http.StatusInternalServerError)
			return
		}
		b.WriteTo(w)

	})

	articleRoutes := make(map[string]http.Handler)

	m.Handle("/article/{id}", methodDispatcher(articleRoutes))
	m.HandleFunc("/article", func(w http.ResponseWriter, r *http.Request) {
		ct := r.Header.Get("Content-Type")
		if ct != "application/json" {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		var article Article
		err := json.NewDecoder(r.Body).Decode(&article)
		r.Body.Close()
		if err != nil {
			http.Error(w, fmt.Sprintf("could not decode json: %v", err), http.StatusBadRequest)
			return
		}
		ctx := r.Context()
		if err := s.Create(ctx, article); err != nil {
			http.Error(w, fmt.Sprintf("fail to create: %v", err), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
	})

	articleRoutes[http.MethodGet] = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		if id == "" {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		ctx := r.Context()
		a, err := s.Get(ctx, id)
		if err != nil {
			http.Error(w, fmt.Sprintf("could not read id: %v", err), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(a)
	})

	articleRoutes[http.MethodDelete] = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		if id == "" {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		ctx := r.Context()
		if err := s.Delete(ctx, id); err != nil {
			http.Error(w, "error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}

type methodDispatcher map[string]http.Handler

func (mux methodDispatcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h, ok := mux[r.Method]; ok {
		h.ServeHTTP(w, r)
		return
	}

	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}
