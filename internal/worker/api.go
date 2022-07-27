package worker

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/elimt/go-orchestrator/internal/task"
	"github.com/go-chi/chi"
	"github.com/google/uuid"
)

type API struct {
	Address string
	Port    int
	Worker  *Worker
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
	a.Router.Route("/stats", func(r chi.Router) {
		r.Get("/", a.GetStatsHandler)
	})
}

func (a *API) StartTaskHandler(w http.ResponseWriter, r *http.Request) {
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()

	te := task.TaskEvent{}
	err := d.Decode(&te)
	if err != nil {
		msg := fmt.Sprintf("Error unmarshalling body: %v\n", err)
		fmt.Printf("decoding error: %v", msg)
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

	a.Worker.AddTask(te.Task)
	fmt.Printf("Added task %v\n", te.Task.ID)
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(te.Task)
	if err != nil {
		fmt.Printf("Error encoding error response: %v\n", err)
	}
}

func (a *API) GetTasksHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	err := json.NewEncoder(w).Encode(a.Worker.GetTasks())
	if err != nil {
		fmt.Printf("Error encoding error response: %v\n", err)
	}
}

func (a *API) StopTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	if taskID == "" {
		fmt.Printf("No taskID passed in request.\n")
		w.WriteHeader(400)
	}

	tID, _ := uuid.Parse(taskID)
	_, ok := a.Worker.DB[tID]
	if !ok {
		fmt.Printf("No task with ID %v found", tID)
		w.WriteHeader(404)
	}

	taskToStop := a.Worker.DB[tID]
	taskCopy := *taskToStop
	taskCopy.State = task.Completed
	a.Worker.AddTask(taskCopy)

	fmt.Printf("Added task %v to stop container %v\n", taskToStop.ID, taskToStop.ContainerID)
	w.WriteHeader(204)
}

func (a *API) GetStatsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	err := json.NewEncoder(w).Encode(a.Worker.Stats)
	if err != nil {
		fmt.Printf("Error encoding error response: %v\n", err)
	}
}
