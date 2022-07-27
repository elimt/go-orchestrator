package manager

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/elimt/go-orchestrator/internal/task"
	"github.com/go-chi/chi"
	"github.com/google/uuid"
)

type API struct {
	Address string
	Port    int
	Manager *Manager
	Router  *chi.Mux
}

type ErrResponse struct {
	HTTPStatusCode int
	Message        string
}

func (a *API) Start() {
	a.initRouter()
	err := http.ListenAndServe(fmt.Sprintf("%s:%d", a.Address, a.Port), a.Router)
	if err != nil {
		fmt.Printf("Error starting worker http server: %v\n", err)
		panic(err)
	}
}

func (a *API) initRouter() {
	a.Router = chi.NewRouter()
	a.Router.Route("/tasks", func(r chi.Router) {
		r.Post("/", a.StartTaskHandler)
		r.Get("/", a.GetTasksHandler)
		r.Route("/{taskID}", func(r chi.Router) {
			r.Delete("/", a.StopTaskHandler)
		})
	})
}

func (a *API) StartTaskHandler(w http.ResponseWriter, r *http.Request) {
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()

	te := task.TaskEvent{}
	err := d.Decode(&te)
	if err != nil {
		msg := fmt.Sprintf("Error unmarshalling body: %v\n", err)
		fmt.Printf("error: %v", msg)
		w.WriteHeader(http.StatusBadRequest)
		e := ErrResponse{
			HTTPStatusCode: http.StatusBadRequest,
			Message:        msg,
		}
		err = json.NewEncoder(w).Encode(e)
		if err != nil {
			fmt.Printf("Error encoding error response: %v\n", err)
		}
		return
	}

	a.Manager.AddTask(te)
	fmt.Printf("Added task %v\n", te.Task.ID)
	w.WriteHeader(201)
	err = json.NewEncoder(w).Encode(te.Task)
	if err != nil {
		fmt.Printf("Error encoding error response: %v\n", err)
	}
}

func (a *API) GetTasksHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(a.Manager.GetTasks())
	if err != nil {
		fmt.Printf("Error encoding error response: %v\n", err)
	}
}

func (a *API) StopTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	if taskID == "" {
		fmt.Printf("No taskID passed in request.\n")
		w.WriteHeader(http.StatusBadRequest)
	}

	tID, _ := uuid.Parse(taskID)
	_, ok := a.Manager.TaskDB[tID]
	if !ok {
		fmt.Printf("No task with ID %v found", tID)
		w.WriteHeader(404)
	}

	te := task.TaskEvent{
		ID:        uuid.New(),
		State:     task.Completed,
		Timestamp: time.Now(),
	}
	taskToStop := a.Manager.TaskDB[tID]
	// we need to make a copy so we are not modifying the task in the datastore
	taskCopy := *taskToStop
	taskCopy.State = task.Completed
	te.Task = taskCopy
	a.Manager.AddTask(te)

	fmt.Printf("Added task event %v to stop task %v\n", te.ID, taskToStop.ID)
	w.WriteHeader(http.StatusNoContent)
}
