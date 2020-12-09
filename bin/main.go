package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"

	"example.com/service"

	_ "github.com/proullon/ramsql/driver"
)

func main() {

	db, err := sql.Open("ramsql", "somewhere")
	if err != nil {
		log.Fatalf("could not open database: %s\n", err)
	}
	defer db.Close()

	svc := &service.ArticleService{DB: db}

	svc.Prepare(context.TODO())
	log.Println("start running service")

	http.Handle("/api/", http.StripPrefix("/api", svc.RESTful()))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
