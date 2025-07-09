package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/meghashyamc/wheresthat/db/kvdb"
	"github.com/meghashyamc/wheresthat/db/searchdb"
	"github.com/meghashyamc/wheresthat/logger"
	"github.com/meghashyamc/wheresthat/validation"
)

type server struct {
	router     *gin.Engine
	httpServer *http.Server
	kvdb       kvdb.DB
	searchdb   searchdb.DB
	validator  *validation.Validator
	logger     logger.Logger
}

func Run(ctx context.Context) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)

	defer cancel()

	s := &server{
		logger: logger.New(),
	}
	if err := s.setupDependencies(); err != nil {
		return err
	}
	s.setupRouter()
	s.setupHTTPServer()
	s.setupGracefulShutdown(ctx)

	return nil
}

func (s *server) setupDependencies() error {
	var err error
	s.kvdb, err = kvdb.New(s.logger)
	if err != nil {
		s.logger.Error("error creating kvDB", "err", err.Error())
		return err
	}
	s.searchdb, err = searchdb.New(s.logger)
	if err != nil {
		s.logger.Error("error creating searchDB", "err", err.Error())
		return err
	}
	s.validator, err = validation.New(s.logger)
	if err != nil {
		s.logger.Error("error creating validator", "err", err.Error())
		return err
	}

	return nil

}

func (s *server) setupRouter() {
	router := newRouter()

	router.Use(loggingMiddleware(s.logger))

	setupRoutes(router, s.logger, s.searchdb, s.kvdb, s.validator)

	s.router = router
}

func (s *server) setupHTTPServer() {

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%s", os.Getenv("PORT")),
		Handler: s.router.Handler(),
	}
	s.httpServer = httpServer
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()
}

func (s *server) setupGracefulShutdown(ctx context.Context) {

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		s.logger.Info("starting to shut down http server")
		shutdownCtx := context.Background()
		shutdownCtx, cancel := context.WithTimeout(shutdownCtx, 10*time.Second)
		defer cancel()
		s.kvdb.Close()
		s.searchdb.Close()
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("error shutting down http server", "err", err)
			return
		}
		s.logger.Info("shut down http server successfully")
	}()

	wg.Wait()
}
