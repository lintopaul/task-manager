// main function for task-manager
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/lintopaul/task-manager/handlers"
)

const (
	// timeout for context
	timeout = 30
)

func main() {
	logger := log.New(os.Stdout, "task-manager-api ", log.LstdFlags)

	var wg sync.WaitGroup

	router := mux.NewRouter()

	taskHandler := handlers.NewTaskHandler(logger, &wg)

	router.HandleFunc("/create", taskHandler.CreateTask).Methods("GET")

	api := router.PathPrefix("/").Subrouter()
	api.Use(taskHandler.MiddlewareCheckTask)

	api.HandleFunc("/pause/{id}", taskHandler.PauseTask).Methods("GET")
	api.HandleFunc("/delete/{id}", taskHandler.DeleteTask).Methods("GET")
	api.HandleFunc("/resume/{id}", taskHandler.ResumeTask).Methods("GET")

	s := &http.Server{
		Addr:    ":9090",
		Handler: router,
	}

	go func() {
		logger.Println("Starting server on port 9090")
		logger.Fatal(s.ListenAndServe())
	}()

	// listen for os interrupt and kill commands on server
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT)
	signal.Notify(sigChan, syscall.SIGTERM)

	sig := <-sigChan

	// send kill signal to all tasks so that they can rollback gracefully before server is closed
	taskHandler.TerminateTasks()

	// Used such that the go program waits for all the goroutines to finish before it closes
	wg.Wait()

	logger.Println("Received terminate, properly terminated all tasks")
	logger.Println("Reason:", sig)

	tc, cancel := context.WithTimeout(context.Background(), timeout*time.Second)
	defer cancel()

	if err := s.Shutdown(tc); err != nil {
		logger.Println("error: ", err)
	}
}
