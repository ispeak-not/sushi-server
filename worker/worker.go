package worker

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sushi/utils/DB"
	"sushi/utils/config"

	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

type Worker struct {
	config  *config.Config
	log     *logrus.Logger
	cron    *cron.Cron
	handler *Handler
	db      *DB.DB
}

func CreateServer() *http.Server {

	conf, err := config.NewConfig()
	if err != nil {
		panic(any("error reading config.yaml, " + err.Error()))
	}

	log := logrus.New()
	log.Out = os.Stdout
	log.Level = conf.LogLevel()

	if conf.LogFileLocation() == "" {
		log.Fatal("missing log_file_location config.yaml variable")
	}
	logfile, err := os.OpenFile(conf.LogFileLocation(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("failed to open file for logging")
	} else {
		log.Out = logfile
		log.Formatter = &logrus.JSONFormatter{}
	}

	/*
		Initialize Server
	*/
	svr := NewWorker(conf, log)

	/*
	   Initialize DB
	*/
	svr.db = DB.NewDB_MySQL(log, conf.DBConnectionPath())

	/*
	   Initialize Handler
	*/
	svr.handler = NewHandler(svr)

	/*
		Initialize Cron
	*/
	cron := cron.New()
	svr.cron = NewJob(cron, svr.handler)
	svr.cron.Start()

	addr := fmt.Sprintf("%s:%d", "0.0.0.0", conf.WorkerPort())
	httpServer := makeHttpServer(addr)
	return httpServer
}

func NewJob(cron *cron.Cron, handler *Handler) *cron.Cron {
	fmt.Println("Cron job crawl nfts every run on", handler.conf.SpecSchedule())
	cron.AddFunc(handler.conf.SpecSchedule(), handler.GetOwnersForContract)
	handler.CrawlFromWeb3()
	return cron
}

func NewWorker(conf *config.Config, log *logrus.Logger) *Worker {
	return &Worker{
		config: conf,
		log:    log,
	}
}

func makeHttpServer(addr string) *http.Server {
	return &http.Server{
		Addr: addr,
	}
}

func Start() error {
	log.Println("worker server creating...")
	srv := CreateServer()
	log.Println("worker server starting...")
	err := srv.ListenAndServe()
	if err == http.ErrServerClosed {
		log.Println("server shutting down gracefully...")
	} else {
		log.Println("unexpected server shutdown...")
		log.Println("ERR: ", err)
	}
	return err
}
