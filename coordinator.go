package mr

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sync"
	"time"
)

type Task struct {
	TaskID    int
	TaskType  string // "Map" or "Reduce"
	FileName  string
	NReduce   int
	Status    string // "Idle", "InProgress", "Completed"
	StartTime time.Time
}

type Coordinator struct {
	mu         sync.Mutex
	mapTasks   []Task
	reduceTasks []Task
	nReduce    int
	mapPhase   bool
}

func (c *Coordinator) GetTask(args *struct{}, reply *Task) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()

	if c.mapPhase {
		// Assign Map tasks
		for i, task := range c.mapTasks {
			if task.Status == "Idle" {
				c.mapTasks[i].Status = "InProgress"
				c.mapTasks[i].StartTime = now
				*reply = task
				reply.NReduce = c.nReduce
				return nil
			}
			if task.Status == "InProgress" && now.Sub(task.StartTime) > 10*time.Second {
				log.Printf("Reassigning stale map task: %d", task.TaskID)
				c.mapTasks[i].Status = "Idle"
			}
		}
	} else {
		// Assign Reduce tasks
		for i, task := range c.reduceTasks {
			if task.Status == "Idle" {
				c.reduceTasks[i].Status = "InProgress"
				c.reduceTasks[i].StartTime = now
				*reply = task
				reply.NReduce = c.nReduce
				return nil
			}
			if task.Status == "InProgress" && now.Sub(task.StartTime) > 10*time.Second {
				log.Printf("Reassigning stale reduce task: %d", task.TaskID)
				c.reduceTasks[i].Status = "Idle"
			}
		}
	}

	reply.TaskType = "Wait"
	return nil
}

func (c *Coordinator) ReportCompletion(args *Task, reply *struct{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if args.TaskType == "Map" {
		c.mapTasks[args.TaskID].Status = "Completed"
	} else if args.TaskType == "Reduce" {
		c.reduceTasks[args.TaskID].Status = "Completed"
	}

	// Move to Reduce phase if all Map tasks are completed
	if c.mapPhase && allTasksCompleted(c.mapTasks) {
		log.Println("All Map tasks completed. Moving to Reduce phase.")
		c.mapPhase = false
	}
	return nil
}

func allTasksCompleted(tasks []Task) bool {
	for _, task := range tasks {
		if task.Status != "Completed" {
			return false
		}
	}
	return true
}

func (c *Coordinator) Done() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return !c.mapPhase && allTasksCompleted(c.reduceTasks)
}

func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	sockname := coordinatorSock()
	os.Remove(sockname) // Remove any existing socket file
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	log.Printf("Coordinator is listening on %s", sockname)
	go http.Serve(l, nil)
}

func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{
		mapTasks:   make([]Task, len(files)),
		reduceTasks: make([]Task, nReduce),
		nReduce:    nReduce,
		mapPhase:   true,
	}

	// Initialize Map tasks
	for i, file := range files {
		c.mapTasks[i] = Task{
			TaskID:   i,
			TaskType: "Map",
			FileName: file,
			Status:   "Idle",
		}
	}

	// Initialize Reduce tasks
	for i := 0; i < nReduce; i++ {
		c.reduceTasks[i] = Task{
			TaskID:   i,
			TaskType: "Reduce",
			Status:   "Idle",
		}
	}

	c.server()
	return &c
}
