package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/bigkevmcd/micromastro/utils"

	"github.com/jmoiron/sqlx"
	"github.com/rcrowley/go-metrics"
	"github.com/rcrowley/go-tigertonic"
	// We only need this for the registration
	_ "github.com/lib/pq"
)

// TODO Change Handlers to be structs that have access to database and metric.
var connection *sqlx.DB

var (
	port      = utils.GetEnvWithDefault("MASTRO_PROJECTS_PORT", ":8081")
	dsn       = utils.GetEnvWithDefault("MASTRO_PROJECTS_DSN", "dbname=capomastro sslmode=disable")
	reservoir = utils.GetEnvWithDefault("MASTRO_SAMPLE_RESERVOIR", "1024")
)

type Project struct {
	Id          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func ProjectIndex(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	projects := []Project{}
	connection.Select(&projects, "SELECT id, name, description FROM projects_project")

	g := metrics.GetOrRegisterHistogram("projects.get.count", nil, nil)
	g.Update(int64(len(projects)))

	encoder := json.NewEncoder(w)
	encoder.Encode(projects)
}

func main() {
	reservoirSize, err := strconv.Atoi(reservoir)
	if err != nil {
		log.Fatal("MASTRO_SAMPLE_RESERVOIR must be a valid integer")
	}

	go metrics.Log(metrics.DefaultRegistry, 60e9, log.New(os.Stderr, "metrics: ", log.Lmicroseconds))
	metrics.NewRegisteredHistogram("projects.get.count", nil, metrics.NewUniformSample(reservoirSize))
	connection, err = sqlx.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}

	mux := tigertonic.NewTrieServeMux()
	mux.Handle("GET", "/projects", tigertonic.Timed(http.HandlerFunc(ProjectIndex), "projects.get", nil))

	log.Fatal(http.ListenAndServe(port, mux))
}
