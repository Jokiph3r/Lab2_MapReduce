package mr

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"net/rpc"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"time"
)

type KeyValue struct {
	Key   string
	Value string
}

func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

func Worker(mapf func(string, string) []KeyValue, reducef func(string, []string) string) {
	for {
		task := Task{}
		success := call("Coordinator.GetTask", &struct{}{}, &task)
		if !success {
			log.Println("Coordinator unreachable. Retrying in 1 second...")
			time.Sleep(time.Second)
			continue
		}

		if task.TaskType == "Map" {
			log.Printf("Processing Map task: %+v\n", task)
			processMapTask(task, mapf)
		} else if task.TaskType == "Reduce" {
			log.Printf("Processing Reduce task: %+v\n", task)
			processReduceTask(task, reducef)
		} else if task.TaskType == "Wait" {
			log.Println("No task available. Waiting...")
			time.Sleep(time.Second)
			continue
		} else {
			log.Println("Unknown task type. Exiting...")
			return
		}

		call("Coordinator.ReportCompletion", &task, &struct{}{})
		log.Printf("Completed task: %+v\n", task)
	}
}

func processMapTask(task Task, mapf func(string, string) []KeyValue) {
	content, err := ioutil.ReadFile(task.FileName)
	if err != nil {
		log.Fatalf("Failed to read file %s: %v", task.FileName, err)
	}

	kva := mapf(task.FileName, string(content))
	nReduce := task.NReduce
	intermediate := make([][]KeyValue, nReduce)
	for _, kv := range kva {
		reduceBucket := ihash(kv.Key) % nReduce
		intermediate[reduceBucket] = append(intermediate[reduceBucket], kv)
	}

	for i := 0; i < nReduce; i++ {
		fileName := fmt.Sprintf("mr-%d-%d", task.TaskID, i)
		tempFile, err := ioutil.TempFile(".", fileName)
		if err != nil {
			log.Fatalf("Failed to create intermediate file: %v", err)
		}
		enc := json.NewEncoder(tempFile)
		for _, kv := range intermediate[i] {
			if err := enc.Encode(&kv); err != nil {
				log.Fatalf("Failed to write to intermediate file: %v", err)
			}
		}
		tempFile.Close()
		os.Rename(tempFile.Name(), fileName)
	}
}

func processReduceTask(task Task, reducef func(string, []string) string) {
	intermediate := []KeyValue{}

	for i := 0; i < task.NReduce; i++ {
		fileName := fmt.Sprintf("mr-%d-%d", i, task.TaskID)
		file, err := os.Open(fileName)
		if err != nil {
			log.Printf("Warning: Missing intermediate file %s, skipping...", fileName)
			continue
		}

		dec := json.NewDecoder(file)
		for {
			var kv KeyValue
			if err := dec.Decode(&kv); err != nil {
				break
			}
			intermediate = append(intermediate, kv)
		}
		file.Close()
	}

	sort.Slice(intermediate, func(i, j int) bool {
		return intermediate[i].Key < intermediate[j].Key
	})

	output := make(map[string][]string)
	for _, kv := range intermediate {
		output[kv.Key] = append(output[kv.Key], kv.Value)
	}

	outputFile, err := os.Create(fmt.Sprintf("mr-out-%d", task.TaskID))
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	for key, values := range output {
		result := reducef(key, values)
		fmt.Fprintf(outputFile, "%v %v\n", key, result)
	}
	outputFile.Close()
}

func call(rpcname string, args interface{}, reply interface{}) bool {
	sockname := coordinatorSock()
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}
