package worker

import (
	"errors"
	"fmt"
	"time"

	"github.com/elimt/go-orchestrator/internal/task"
	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
)

type Worker struct {
	Name      string
	Queue     queue.Queue
	DB        map[uuid.UUID]*task.Task
	TaskCount int
	Stats     *Stats
}

func (w *Worker) CollectStats() {
	for {
		fmt.Println("Collecting stats")
		w.Stats = GetStats()
		w.TaskCount = w.Stats.TaskCount
		time.Sleep(15 * time.Second)
	}
}

func (w *Worker) runTask() task.DockerResult {
	t := w.Queue.Dequeue()
	if t == nil {
		fmt.Println("No tasks in the queue")
		return task.DockerResult{Error: nil}
	}

	taskQueued := t.(task.Task)

	taskPersisted := w.DB[taskQueued.ID]
	if taskPersisted == nil {
		taskPersisted = &taskQueued
		w.DB[taskQueued.ID] = &taskQueued
	}

	var result task.DockerResult
	if task.ValidStateTransition(taskPersisted.State, taskQueued.State) {
		switch taskQueued.State {
		case task.Scheduled:
			result = w.StartTask(taskQueued)
		case task.Completed:
			result = w.StopTask(taskQueued)
		default:
			result.Error = errors.New("we should not get here")
		}
	} else {
		err := fmt.Errorf("invalid transition from %v to %v", taskPersisted.State, taskQueued.State)
		result.Error = err
	}
	return result
}

func (w *Worker) RunTasks() {
	for {
		if w.Queue.Len() != 0 {
			result := w.runTask()
			if result.Error != nil {
				fmt.Printf("Error running task: %v\n", result.Error)
			}
		} else {
			fmt.Printf("No tasks to process currently.\n")
		}
		fmt.Println("Sleeping for 10 seconds.")
		time.Sleep(10 * time.Second)
	}

}

func (w *Worker) AddTask(t task.Task) {
	w.Queue.Enqueue(t)
}

func (w *Worker) StartTask(t task.Task) task.DockerResult {
	fmt.Println("I will start a task")
	config := task.NewConfig(&t)
	d := task.NewDocker(config)
	result := d.Run()
	if result.Error != nil {
		fmt.Printf("Err running task %v: %v\n", t.ID, result.Error)
		t.State = task.Failed
		w.DB[t.ID] = &t
		return result
	}

	d.ContainerID = result.ContainerID
	t.ContainerID = result.ContainerID
	t.State = task.Running
	w.DB[t.ID] = &t

	return result
}

func (w *Worker) StopTask(t task.Task) task.DockerResult {
	fmt.Println("I will stop a task")
	config := task.NewConfig(&t)
	d := task.NewDocker(config)

	result := d.Stop()
	if result.Error != nil {
		fmt.Printf("Error stopping container %v: %v", d.ContainerID, result.Error)
	}
	t.FinishTime = time.Now().UTC()
	t.State = task.Completed
	w.DB[t.ID] = &t
	fmt.Printf("Stopped and removed container %v for task %v", d.ContainerID, t.ID)

	return result
}

func (w *Worker) GetTasks() []*task.Task {
	tasks := make([]*task.Task, 0)
	for _, t := range w.DB {
		tasks = append(tasks, t)
	}
	return tasks
}
