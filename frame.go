package frame

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	gfbus "github.com/greatfocus/gf-bus"
	gfcron "github.com/greatfocus/gf-cron"
	"github.com/greatfocus/gf-sframe/broker"
	"github.com/greatfocus/gf-sframe/cache"
	"github.com/greatfocus/gf-sframe/database"
	"github.com/greatfocus/gf-sframe/logger"
	"github.com/greatfocus/gf-sframe/server"
	"github.com/joho/godotenv"
)

// Frame struct
type Frame struct {
	env    string
	Server *server.Meta
}

// NewFrame get new instance of frame
func NewFrame(serviceName string) *Frame {
	// Load environment variables
	env := os.Getenv("ENV")
	if env == "" || os.Getenv("ENV") == "dev" {
		if err := godotenv.Load(".env"); err != nil {
			log.Fatal(err)
		}
	}
	env = os.Getenv("ENV")

	// prepare impl
	impl := Impl{
		Service: serviceName,
		Env:     env,
	}
	var f = &Frame{env: impl.Env}
	f.Server = f.init(&impl)
	return f
}

// Init provides a way to initialize the frame
func (f *Frame) init(impl *Impl) *server.Meta {

	// initCron creates instance of cron
	cron := f.initCron()

	// initDB create database connection
	db := f.initDB(impl)

	// initCache creates instance of cache
	cache := f.initCache()

	// initCron creates instance of cron
	jwt := f.initJWT()

	// create new bus instance
	bus := f.initServiceBus()

	// create new broker instance
	broker := f.initServiceBroker()

	// init creates instance of logger
	logger := f.initLogger(impl.Service)

	return &server.Meta{
		Env:    impl.Env,
		Cron:   cron,
		Cache:  cache,
		DB:     db,
		JWT:    jwt,
		Bus:    bus,
		Logger: logger,
		Broker: broker,
	}
}

// Start spins up the service
func (f *Frame) Start(mux *http.ServeMux) {
	f.Server.Mux = mux
	f.Server.Start()
}

// initCron creates instance of cron
func (f *Frame) initCron() *gfcron.Cron {
	return gfcron.New()
}

func (f *Frame) initCache() *cache.Cache {
	// Create a cache with a default expiration time of 5 minutes, and which
	// purges expired items every 10 minutes
	expireVal, err := strconv.ParseInt(os.Getenv("CACHE_EXPIRE"), 0, 64)
	if err != nil {
		log.Fatal(fmt.Println(err))
	}

	intervalVal, err := strconv.ParseInt(os.Getenv("CACHE_INTERVAL"), 0, 64)
	if err != nil {
		log.Fatal(fmt.Println(err))
	}
	return cache.New(expireVal, intervalVal)
}

// initDB connection
func (f *Frame) initDB(impl *Impl) *database.Conn {
	var db = database.Conn{}
	db.Init()
	return &db
}

// initJWT creates instance of auth
func (f *Frame) initJWT() *server.JWT {
	var jwt = server.JWT{}
	jwt.Init()
	return &jwt
}

// initServiceBus provides bus instance
func (f *Frame) initServiceBus() *gfbus.Bus {
	// create service bus
	bus := gfbus.New()
	return &bus
}

// initServiceBroker provides broker instance
func (f *Frame) initServiceBroker() *broker.Conn {
	conn, err := broker.GetConn(os.Getenv("RABBITMQ_URL"))
	if err != nil {
		panic(err)
	}
	return conn
}

// initServiceBus provides bus instance
func (f *Frame) initLogger(serviceName string) *logger.Logger {
	// create service bus
	return logger.NewLogger(serviceName)
}
