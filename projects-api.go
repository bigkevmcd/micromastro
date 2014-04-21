package main

import (
	"database/sql"
	"log"
	"net/http"
	"encoding/json"

	"github.com/rcrowley/go-metrics"
	"github.com/rcrowley/go-tigertonic"
	// We only need this for the registration
	_ "github.com/lib/pq"
)

var connection *sql.DB

var (
	port = utils.GetEnvWithDefault("MASTRO_PROJECTS_PORT", ":8081")
	dsn  = utils.GetEnvWithDefault("MASTRO_PROJECTS_DSN", "dbname=capomastro sslmode=disable")
)

type Project struct {
	Id          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func ProjectIndex(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	result := make([]Project, 0)
	rows, err := connection.Query("SELECT id, name, description FROM projects_project")
	if err != nil {
		log.Println(err)
		return
	}

	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.Id, &p.Name, &p.Description); err != nil {
			log.Println(err)
			return
		}
		result = append(result, p)
	}
	if err := rows.Err(); err != nil {
		log.Println(err)
		return
	}

	encoder := json.NewEncoder(w)
	encoder.Encode(result)
}

func main() {
	var err error
	connection, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}

	mux := tigertonic.NewTrieServeMux()
	mux.HandleFunc("GET", "/projects", ProjectIndex)

	log.Fatal(http.ListenAndServe(port, mux))
}
