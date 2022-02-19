package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	gfbus "github.com/greatfocus/gf-bus"
	gfcron "github.com/greatfocus/gf-cron"
	"github.com/greatfocus/gf-sframe/cache"
	"github.com/greatfocus/gf-sframe/database"
)

// HandlerFunc custom server handler
type HandlerFunc func(http.ResponseWriter, *http.Request)

// Meta struct
type Meta struct {
	Env   string
	Mux   *http.ServeMux
	DB    *database.Conn
	Cache *cache.Cache
	Cron  *gfcron.Cron
	JWT   *JWT
	Bus   *gfbus.Bus
}

// Start the server
func (m *Meta) Start() {
	// setUploadPath creates an upload path
	m.setUploadPath()

	// serve creates server instance
	m.serve()
}

// setUploadPath creates an upload path
func (m *Meta) setUploadPath() {
	uploadPath := os.Getenv("ENV")
	if uploadPath != "" {
		fs := http.FileServer(http.Dir(uploadPath + "/"))
		m.Mux.Handle("/file/", http.StripPrefix("/file/", fs))
	}
}

// serve creates server instance
func (m *Meta) serve() {
	timeout, err := strconv.ParseUint(os.Getenv("SERVER_TIMEOUT"), 0, 64)
	if err != nil {
		log.Fatal(fmt.Println(err))
	}

	addr := ":" + os.Getenv("SERVER_PORT")
	srv := &http.Server{
		Addr:           addr,
		ReadTimeout:    time.Duration(timeout) * time.Second,
		WriteTimeout:   time.Duration(timeout) * time.Second,
		MaxHeaderBytes: 1 << 20,
		Handler:        m.Mux,
	}

	// create server connection
	log.Println("Listening to port HTTP", addr)
	log.Fatal(srv.ListenAndServe())
}
