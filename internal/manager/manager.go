package manager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/elimt/go-orchestrator/internal/task"
	"github.com/elimt/go-orchestrator/internal/worker"
	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
)

type Manager struct {
	Pending       queue.Queue
	TaskDB        map[uuid.UUID]*task.Task
	EventDB       map[uuid.UUID]*task.TaskEvent
	Workers       []string
	WorkerTaskMap map[string][]uuid.UUID
	TaskWorkerMap map[uuid.UUID]string
	LastWorker    int
}

func New(workers []string) *Manager {
	taskDB := make(map[uuid.UUID]*task.Task)
	eventDB := make(map[uuid.UUID]*task.TaskEvent)
	workerTaskMap := make(map[string][]uuid.UUID)
	taskWorkerMap := make(map[uuid.UUID]string)
	for worker := range workers {
		workerTaskMap[workers[worker]] = []uuid.UUID{}
	}

	return &Manager{
		Pending:       *queue.New(),
		Workers:       workers,
		TaskDB:        taskDB,
		EventDB:       eventDB,
		WorkerTaskMap: workerTaskMap,
		TaskWorkerMap: taskWorkerMap,
	}
}

func (m *Manager) AddTask(te task.TaskEvent) {
	m.Pending.Enqueue(te)
}

func (m *Manager) updateTasks() {
	for _, worker := range m.Workers {
		fmt.Printf("Checking worker %v for task updates", worker)
		url := fmt.Sprintf("http://%s/tasks", worker)
		//nolint:gosec
		resp, err := http.Get(url)
		if err != nil {
			fmt.Printf("Error connecting to %v: %v", worker, err)
		}

		if resp.StatusCode != http.StatusOK {
			fmt.Printf("Error sending request: %v", err)
		}

		d := json.NewDecoder(resp.Body)
		var tasks []*task.Task
		err = d.Decode(&tasks)
		if err != nil {
			fmt.Printf("Error unmarshalling tasks: %s", err.Error())
		}

		for _, t := range tasks {
			fmt.Printf("Attempting to update task %v", t.ID)

			_, ok := m.TaskDB[t.ID]
			if !ok {
				fmt.Printf("Task with ID %s not found\n", t.ID)
				return
			}

			if m.TaskDB[t.ID].State != t.State {
				m.TaskDB[t.ID].State = t.State
			}

			m.TaskDB[t.ID].StartTime = t.StartTime
			m.TaskDB[t.ID].FinishTime = t.FinishTime
			m.TaskDB[t.ID].ContainerID = t.ContainerID
		}
	}
}

func (m *Manager) UpdateTasks() {
	for {
		fmt.Println("Checking for task updates from workers")
		m.updateTasks()
		fmt.Println("Task updates completed")
		fmt.Println("Sleeping for 15 seconds")
		time.Sleep(15 * time.Second)
	}
}

func (m *Manager) ProcessTasks() {
	for {
		fmt.Println("Processing any tasks in the queue")
		m.SendWork()
		fmt.Println("Sleeping for 10 seconds")
		time.Sleep(10 * time.Second)
	}
}

func (m *Manager) SendWork() {
	fmt.Println("I will send work to workers")
	if m.Pending.Len() > 0 {
		w := m.SelectWorker()

		e := m.Pending.Dequeue()
		te := e.(task.TaskEvent)
		t := te.Task
		fmt.Printf("Pulled %v off pending queue", t)

		m.EventDB[te.ID] = &te
		m.WorkerTaskMap[w] = append(m.WorkerTaskMap[w], te.Task.ID)
		m.TaskWorkerMap[t.ID] = w

		t.State = task.Scheduled
		m.TaskDB[t.ID] = &t

		data, err := json.Marshal(te)
		if err != nil {
			fmt.Printf("Unable to marshal task object: %v.", t)
		}

		url := fmt.Sprintf("http://%s/tasks", w)
		//nolint:gosec
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
		if err != nil {
			fmt.Printf("Error connecting to %v: %v", w, err)
			m.Pending.Enqueue(t)
			return
		}

		d := json.NewDecoder(resp.Body)
		if resp.StatusCode != http.StatusCreated {
			e := worker.ErrResponse{}
			err := d.Decode(&e)
			if err != nil {
				fmt.Printf("Error decoding response: %s\n", err.Error())
				return
			}
			fmt.Printf("Response error (%d): %s", e.HTTPStatusCode, e.Message)
			return
		}

		t = task.Task{}
		err = d.Decode(&t)
		if err != nil {
			fmt.Printf("Error decoding response: %s\n", err.Error())
			return
		}
		fmt.Printf("%#v\n", t)
	} else {
		fmt.Println("No work in the queue")
	}
}

// SelectWorker using a round robin algorithm
func (m *Manager) SelectWorker() string {
	fmt.Println("I will send work to workers")
	var newWorker int
	if m.LastWorker+1 < len(m.Workers) {
		newWorker = m.LastWorker + 1
		m.LastWorker++
	} else {
		newWorker = 0
		m.LastWorker = 0
	}

	return m.Workers[newWorker]
}

func (m *Manager) GetTasks() []*task.Task {
	tasks := make([]*task.Task, len(m.TaskWorkerMap))
	for t := range m.TaskWorkerMap {
		tasks = append(tasks, m.TaskDB[t])
	}
	return tasks
}
