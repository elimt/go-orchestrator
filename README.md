# go-orchestrator
Repository for working through [Build An Orchestrator in Go](https://www.manning.com/books/build-an-orchestrator-in-go)

## Details
* Go Version: 1.18

## Running Locally
1. Start up application with `task start`
1. Add tasks by firing REST Call 
```bash
curl -v --request POST \                                                            
  --header 'Content-Type: application/json' \
  --data '{
    "ID": "266592cd-960d-4091-981c-8c25c44b1018",
    "State": 2,
    "Task": {
        "State": 1,
        "ID": "266592cd-960d-4091-981c-8c25c44b1018",
        "Name": "task-1",
        "Image": "strm/helloworld-http"
    }
}
```
3. List all tasks
```bash
curl http://localhost:8989/tasks |jq .
```
4. Delete task
```bash
curl -v --request DELETE "localhost:8989/tasks/75e260da-e9f7-4601-bca2-d52461df12cc"
```


### Task File

Task is a task runner / build tool that aims to be simply common commands. The commands can be found in the [Taskfile.yml](Taskfile.yml) file.

#### Task File Installation

Installation instructions for Task can be found [here](https://taskfile.dev/#/installation?id=get-the-binary).

#### List Task Commands

To list all available commands for this project:

```bash
$ task --list
```

## Definitions 
### Orchestrator
  * Automates deploying, scaling & managing containers.
### Task
  * Smallest unit of work. 
    * Should specify it's needs for memory, CPU & disk. 
    * What to do in case of failure (restart_policy). 
    * Name of container image used to run the task.
### Job (not included in this implementation)
  * Group of tasks to perform a set of functions. 
  * Should specify details at a higher level than tasks. 
    * Each task that makes up the job. 
    * Which data centers the job should run in. 
    * The type of job. 
### Scheduler
  * Decides which machine can best host the tasks defined. 
  * Could be a process within the machine or possibly a different service.
  * Decisions can vary in model complexity from round-robin to score calculation based on variables and deciding node by best score 
    * Determine set of candidate machines to run task
    * Score machines from best to worst
    * Pick machine with best score
### Manager 
  * Brain of orchestrator & main entry point
  * Jobs go to manager, manager uses scheduler to determine where to run job. 
  * Can gather metrics, to use for scheduling.
    * Accept requests from users to start/stop tasks
    * Schedule tasks onto worker machines
    * Keep track of tasks, states & the machine they run on 
### Worker
  * Runs the tasks assigned to it, attempts task restarts on failure, makes metrics about task and machine health for master
    * Runs asks as containers
    * Accepts tasks to run from manager
    * provides relevant statistics to the manager to schedule tasks
    * Keeps track of tasks & their state 
### Cluster
  * Logical grouping of all above components. 
### CLI 
  * Main user interface
    * Start/Stop Tasks
    * Get status of tasks
    * See state of machines
    * Start manager
    * Start worker 