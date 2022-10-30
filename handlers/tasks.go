// Package handlers handles the routes
package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

const (
	// Constants denoting the state of a task
	start = 2
	pause = 1
	kill  = 0

	// taskDuration indicates the duration of the task to be simulated
	taskDuration = 3
	// taskCount indicates the number of tasks
	taskCount = 10
	// rollbackDuration
	rollbackDuration = 1
)

// Response represents the API response
type Response struct {
	UUID    string `json:"uuid,omitempty"`
	Err     string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
	Success bool   `json:"success"`
}

// responseWriter sends the response to client in the form of json
func responseWriter(w http.ResponseWriter, payload interface{}, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Println("error: ", err)
	}
}

var counter int

// TaskHandler handles task handling requests
type TaskHandler struct {
	logger  *log.Logger
	wg      *sync.WaitGroup
	workers map[string](chan int)
	states  map[string]int
}

// NewTaskHandler creates a new instance of TaskHandler
func NewTaskHandler(l *log.Logger, wg *sync.WaitGroup) *TaskHandler {
	workers := make(map[string](chan int))
	states := make(map[string]int)
	return &TaskHandler{l, wg, workers, states}
}

// CreateTask spawns a new task
func (t *TaskHandler) CreateTask(w http.ResponseWriter, _ *http.Request) {
	t.logger.Println("Endpoint: create")
	rawUUID := uuid.New()
	uuid := strings.ReplaceAll(rawUUID.String(), "-", "")
	t.workers[uuid] = make(chan int, 1)
	t.states[uuid] = start
	go task(counter, uuid, taskCount, t)
	t.workers[uuid] <- start
	t.logger.Println("Task created. uuid:", uuid)
	counter++
	resp := Response{Success: true, UUID: uuid}
	responseWriter(w, resp, http.StatusOK)
}

// PauseTask pauses a task
func (t *TaskHandler) PauseTask(rw http.ResponseWriter, r *http.Request) {
	if uuid, ok := r.Context().Value(KeyUUID{}).(string); ok {
		if t.states[uuid] == pause {
			resp := Response{Success: true, Message: "Already paused"}
			responseWriter(rw, resp, http.StatusOK)
			return
		}
		t.logger.Println("Endpoint: pause")
		t.workers[uuid] <- pause
		t.states[uuid] = pause
		resp := Response{Success: true}
		responseWriter(rw, resp, http.StatusOK)
	}
}

// ResumeTask resumes a paused task
func (t *TaskHandler) ResumeTask(rw http.ResponseWriter, r *http.Request) {
	if uuid, ok := r.Context().Value(KeyUUID{}).(string); ok {
		if t.states[uuid] == start {
			resp := Response{Success: true, Message: "Already runnning"}
			responseWriter(rw, resp, http.StatusOK)
			return
		}

		t.logger.Println("Endpoint: resume")
		t.workers[uuid] <- start
		t.states[uuid] = start
		resp := Response{Success: true}
		responseWriter(rw, resp, http.StatusOK)
	}
}

// DeleteTask terminates an ongoing task
func (t *TaskHandler) DeleteTask(rw http.ResponseWriter, r *http.Request) {
	if uuid, ok := r.Context().Value(KeyUUID{}).(string); ok {
		t.logger.Println("Endpoint: delete")
		t.workers[uuid] <- kill
		t.states[uuid] = kill
		resp := Response{Success: true}
		responseWriter(rw, resp, http.StatusOK)
	}
}

func (t *TaskHandler) closeRoutine(uuid string) {
	t.wg.Done()
	close(t.workers[uuid])
	delete(t.workers, uuid)
	delete(t.states, uuid)
}

// TerminateTasks sends a kill signal to all the running tasks
func (t *TaskHandler) TerminateTasks() {
	for k := range t.workers {
		t.workers[k] <- kill
	}
}

// KeyUUID represents data saved in the context
type KeyUUID struct{}

// MiddlewareCheckTask validates the uuid passed with the endpoint and saves it in the context
func (t *TaskHandler) MiddlewareCheckTask(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		i := vars["id"]

		_, ok := t.workers[i]

		if !ok {
			err := Response{Success: false, Err: "task for given uuid does not exist"}
			responseWriter(rw, err, http.StatusBadRequest)
			return
		}

		ctx := context.WithValue(r.Context(), KeyUUID{}, i)
		r = r.WithContext(ctx)

		next.ServeHTTP(rw, r)
	})
}

// task implements pause, resume and delete functionality
func task(id int, uuid string, count int, t *TaskHandler) {
	t.wg.Add(1)
	defer t.closeRoutine(uuid)

	// simulates long running task
	for i := 0; i < count; i++ {
		if len(t.workers[uuid]) > 0 {
			state := <-t.workers[uuid]
			if state == pause {
				t.logger.Println("uuid:", uuid, "status: paused")
				for state == pause {
					state = <-t.workers[uuid]
				}
			}

			if state == kill {
				t.logger.Println("uuid:", uuid, "status: killed")
				t.logger.Println("rollback initiated")
				go rollBack(uuid, t)
				return
			}

			t.logger.Println("uuid:", uuid, "status: running")
		}
		// ensures concurrency by forcing scheduler to rechedule on another task
		runtime.Gosched()

		// Dummy task, can be replaced with actual task to be performed
		t.logger.Println("id:", id, "value:", i)
		time.Sleep(taskDuration * time.Second)
	}

	t.logger.Println("uuid:", uuid, "status: completed")
}

func rollBack(uuid string, t *TaskHandler) {
	t.wg.Add(1)
	defer t.wg.Done()

	// Dummy task, can be replaced with actual rollback task to be performed
	time.Sleep(rollbackDuration * time.Second)
	t.logger.Println("uuid:", uuid, "status: Rollback completed")
}
