package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"

	_ "main/docs"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/stdlib"
	"github.com/rs/cors"
	httpSwagger "github.com/swaggo/http-swagger"
)

const (
	host     = "rc1b-0awqsvptwdah2yhu.mdb.yandexcloud.net,rc1d-u00mpiexrxzjqknb.mdb.yandexcloud.net"
	port     = 6432
	user     = "user1"
	password = "password"
	dbname   = "db1"
	ca       = "/home/rino/.postgresql/root.crt"
)

var db *sql.DB

type Item struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// @title Go CRUD API
// @version 1.0
// @description This is a simple CRUD API.
// @host 178.154.207.88:8080
// @BasePath /
func main() {
	var err error
	db, err = connectDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	router := mux.NewRouter()
	router.HandleFunc("/items", getItems).Methods("GET")
	router.HandleFunc("/items/{id}", getItem).Methods("GET")
	router.HandleFunc("/items", createItem).Methods("POST")
	router.HandleFunc("/items/{id}", updateItem).Methods("PUT")
	router.HandleFunc("/items/{id}", deleteItem).Methods("DELETE")

	// Serve static files
	fs := http.FileServer(http.Dir("./static"))
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))

	// Serve frontend HTML
	router.PathPrefix("/").HandlerFunc(serveFrontend)

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	router.PathPrefix("/swagger/").Handler(c.Handler(httpSwagger.WrapHandler))

	handler := c.Handler(router)

	portStr := os.Getenv("PORT")
	log.Printf("Server is running on port %s\n", portStr)
	log.Fatal(http.ListenAndServe(":"+portStr, handler))
}

func connectDB() (*sql.DB, error) {
	certPool := x509.NewCertPool()
	pem, err := ioutil.ReadFile(ca)
	if err != nil {
		return nil, fmt.Errorf("unable to read CA file: %v", err)
	}

	if !certPool.AppendCertsFromPEM(pem) {
		return nil, fmt.Errorf("failed to append PEM.")
	}

	config := tls.Config{
		RootCAs: certPool,
	}

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=verify-full", user, password, host, port, dbname)
	config.Certificates = make([]tls.Certificate, 0)
	pgxConfig, err := pgx.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("unable to parse config: %v", err)
	}
	pgxConfig.TLSConfig = &config

	db := stdlib.OpenDB(*pgxConfig)

	return db, db.Ping()
}

// getItems godoc
// @Summary Get all items
// @Description Get all items
// @Tags items
// @Produce json
// @Success 200 {array} Item
// @Failure 500 {string} string "Internal Server Error"
// @Router /items [get]
func getItems(w http.ResponseWriter, r *http.Request) {
	rows, err := db.QueryContext(context.Background(), "SELECT id, name FROM items")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	items := []Item{}
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		items = append(items, item)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

// getItem godoc
// @Summary Get an item by ID
// @Description Get an item by ID
// @Tags items
// @Produce json
// @Param id path int true "Item ID"
// @Success 200 {object} Item
// @Failure 400 {string} string "Invalid ID"
// @Failure 404 {string} string "Not Found"
// @Failure 500 {string} string "Internal Server Error"
// @Router /items/{id} [get]
func getItem(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var item Item
	err = db.QueryRowContext(context.Background(), "SELECT id, name FROM items WHERE id=$1", id).Scan(&item.ID, &item.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

// createItem godoc
// @Summary Create a new item
// @Description Create a new item
// @Tags items
// @Accept json
// @Produce json
// @Param item body Item true "New item"
// @Success 201 {object} Item
// @Failure 400 {string} string "Bad Request"
// @Failure 500 {string} string "Internal Server Error"
// @Router /items [post]
func createItem(w http.ResponseWriter, r *http.Request) {
	var item Item
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := db.QueryRowContext(context.Background(), "INSERT INTO items(name) VALUES($1) RETURNING id", item.Name).Scan(&item.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

// updateItem godoc
// @Summary Update an item by ID
// @Description Update an item by ID
// @Tags items
// @Accept json
// @Produce json
// @Param id path int true "Item ID"
// @Param item body Item true "Updated item"
// @Success 204 {string} string "No Content"
// @Failure 400 {string} string "Invalid ID"
// @Failure 500 {string} string "Internal Server Error"
// @Router /items/{id} [put]
func updateItem(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var item Item
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err = db.ExecContext(context.Background(), "UPDATE items SET name=$1 WHERE id=$2", item.Name, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// deleteItem godoc
// @Summary Delete an item by ID
// @Description Delete an item by ID
// @Tags items
// @Param id path int true "Item ID"
// @Success 204 {string} string "No Content"
// @Failure 400 {string} string "Invalid ID"
// @Failure 500 {string} string "Internal Server Error"
// @Router /items/{id} [delete]
func deleteItem(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	_, err = db.ExecContext(context.Background(), "DELETE FROM items WHERE id=$1", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func serveFrontend(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./static/index.html")
}
