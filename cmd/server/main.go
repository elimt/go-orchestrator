package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/elimt/go-orchestrator/internal/manager"
	"github.com/elimt/go-orchestrator/internal/task"
	"github.com/elimt/go-orchestrator/internal/worker"
	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
)

func main() {
	fmt.Println("Starting Go Orchestrator worker")

	whost := os.Getenv("WORKER_HTTP_HOST")
	wport, _ := strconv.Atoi(os.Getenv("WORKER_HTTP_PORT"))

	mhost := os.Getenv("MANAGER_HTTP_HOST")
	mport, _ := strconv.Atoi(os.Getenv("MANAGER_HTTP_PORT"))

	w := worker.Worker{
		Queue: *queue.New(),
		DB:    make(map[uuid.UUID]*task.Task),
	}
	wapi := worker.API{Address: whost, Port: wport, Worker: &w}

	go w.RunTasks()
	go w.CollectStats()
	go wapi.Start()

	workers := []string{fmt.Sprintf("%s:%d", whost, wport)}
	m := manager.New(workers)
	mapi := manager.API{Address: mhost, Port: mport, Manager: m}

	go m.ProcessTasks()
	go m.UpdateTasks()

	mapi.Start()

	// GenerateTasks(m)
}

// func GenerateTasks(m *manager.Manager) {
// 	for i := 0; i < 3; i++ {
// 		t := task.Task{
// 			ID:    uuid.New(),
// 			Name:  fmt.Sprintf("test-container-%d", i),
// 			State: task.Scheduled,
// 			Image: "strm/helloworld-http",
// 		}
// 		te := task.TaskEvent{
// 			ID:    uuid.New(),
// 			State: task.Running,
// 			Task:  t,
// 		}
// 		m.AddTask(te)
// 		m.SendWork()
// 	}
// }
