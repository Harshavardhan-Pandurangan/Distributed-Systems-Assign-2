package main

import (
    "context"
    "log"
    "io/ioutil"
    "strings"
    "fmt"
    "strconv"
    "os"
    "flag"

    pb "mapreduce/proto/mapreduce"
    "google.golang.org/grpc"
)

var master string

// sumOrdinals returns the sum of the ordinal values of the characters in a string
func sumOrdinals(s string) int {
	sum := 0
	for _, c := range s {
		sum += int(c)
	}
	return sum
}

// Mapper function
func Map() {
    // create a connection to the master
    conn, err := grpc.Dial(master, grpc.WithInsecure())
    if err != nil {
        log.Fatalf("Failed to dial the master: %v", err)
    }
    defer conn.Close()

    // create a client for the master
    client := pb.NewMapReduceServiceClient(conn)

    // request a task from the master
    var task *pb.MapResponse
    for {
        // get a task from the master
        task, err = client.Map(context.Background(), &pb.MapRequest{})
        if err != nil {
            log.Fatalf("Failed to get a task: %v", err)
        }

        // check if the task is done
        if task.WorkerId == -2 {
            log.Printf("Waiting for tasks")
        } else if task.WorkerId == -1 {
            log.Printf("No task available")
            return
        } else {
            break
        }
    }

    // read the input file
    log.Printf("Reading the input file: %s", task.InputFile)
    data, err := ioutil.ReadFile(task.InputFile)
    if err != nil {
        log.Fatalf("Failed to read the input file: %v", err)
    }

    // split the data into words
    words := strings.Fields(string(data))

    // count the words
    wordCount := make(map[string]int)
    for _, word := range words {
        wordCount[word]++
    }

    // hash the words to the number of reducers
    wordHash := make(map[string]int32)
    for word, _ := range wordCount {
        wordHash[word] = int32(sumOrdinals(word)) % task.NumReduceTasks
    }

    // write the intermediate files
    for i := 0; i < int(task.NumReduceTasks); i++ {
        // create the file
        file, err := os.Create(fmt.Sprintf("datasets/intermediate/intermediate-%v-%v", task.WorkerId, i))
        if err != nil {
            log.Fatalf("Failed to create the intermediate file: %v", err)
        }
        defer file.Close()

        // write the data
        if task.TaskId == int32(1) {
            for word, count := range wordCount {
                if int(wordHash[word]) == i {
                    file.WriteString(fmt.Sprintf("%v %v\n", word, count))
                }
            }
        } else {
            for word, hash := range wordHash {
                if hash == int32(i) {
                    file.WriteString(fmt.Sprintf("%v %v\n", word, task.InputFile))
                }
            }
        }
    }

    // send completion update to the master
    res, err := client.Update(context.Background(), &pb.UpdateRequest{
        WorkerId: task.WorkerId,
        WorkerType: "mapper",
        TaskId: int32(task.TaskId),
        Status: "done",
    })
    if err != nil {
        log.Fatalf("Failed to send completion update: %v", err)
    }

    // check if the task is done
    if res.Response != "Success" {
        log.Fatalf("Failed to send completion update: %v", res.Response)
    } else {
        log.Printf("Task %v done", task.TaskId)
    }
}

// Reducer Function
func Reduce() {
    // create a connection to the master
    conn, err := grpc.Dial(master, grpc.WithInsecure())
    if err != nil {
        log.Fatalf("Failed to dial the master: %v", err)
    }
    defer conn.Close()

    // create a client for the master
    client := pb.NewMapReduceServiceClient(conn)

    log.Printf("Reducer started")

    // request a task from the master
    var task *pb.ReduceResponse
    for {
        // get a task from the master
        task, err = client.Reduce(context.Background(), &pb.ReduceRequest{})
        if err != nil {
            log.Fatalf("Failed to get a task: %v", err)
        }

        // check if the task is done
        if task.WorkerId == -2 {
            log.Printf("Waiting for tasks")
        } else if task.WorkerId == -1 {
            log.Printf("No task available")
            return
        } else {
            break
        }
    }

    // read the intermediate files
    intermediateFiles, err := ioutil.ReadDir("datasets/intermediate")
    if err != nil {
        log.Fatalf("Failed to read the intermediate files: %v", err)
    }

    if task.TaskId == int32(1) {
        // read the intermediate files
        wordCount := make(map[string]int)
        for _, file := range intermediateFiles {
            // read the info on the intermediate file
            // split the name of the file at every "-"
            parts := strings.Split(file.Name(), "-")
            if len(parts) != 3 {
                log.Fatalf("Invalid intermediate file name: %v", file.Name())
            }
            if parts[2] != strconv.Itoa(int(task.WorkerId)) {
                continue
            }
            data, err := ioutil.ReadFile(fmt.Sprintf("datasets/intermediate/%v", file.Name()))
            if err != nil {
                log.Fatalf("Failed to read the intermediate file: %v", err)
            }

            // split the data into words
            lines := strings.Split(string(data), "\n")
            for _, line := range lines {
                parts := strings.Fields(line)
                if len(parts) != 2 {
                    continue
                }
                word := parts[0]
                count, err := strconv.Atoi(parts[1])
                if err != nil {
                    log.Fatalf("Failed to convert count to int: %v", err)
                }
                wordCount[word] += count
            }
        }

        // write the output file
        file, err := os.Create(fmt.Sprintf("datasets/output/%v", task.WorkerId))
        if err != nil {
            log.Fatalf("Failed to create the output file: %v", err)
        }
        defer file.Close()

        // write the data
        for word, count := range wordCount {
            file.WriteString(fmt.Sprintf("%v %v\n", word, count))
        }
    } else if task.TaskId == int32(2) {
        // read the intermediate files
        wordFiles := make(map[string][]string)
        for _, file := range intermediateFiles {
            // read the info on the intermediate file
            // split the name of the file at every "-"
            parts := strings.Split(file.Name(), "-")
            if len(parts) != 3 {
                log.Fatalf("Invalid intermediate file name: %v", file.Name())
            }
            if parts[2] != strconv.Itoa(int(task.WorkerId)) {
                continue
            }
            data, err := ioutil.ReadFile(fmt.Sprintf("datasets/intermediate/%v", file.Name()))
            if err != nil {
                log.Fatalf("Failed to read the intermediate file: %v", err)
            }

            // split the data into words
            lines := strings.Split(string(data), "\n")
            for _, line := range lines {
                parts := strings.Fields(line)
                if len(parts) != 2 {
                    continue
                }
                word := parts[0]
                file := parts[1]
                wordFiles[word] = append(wordFiles[word], file)
            }
        }

        // write the output file
        file, err := os.Create(fmt.Sprintf("datasets/output/%v", task.WorkerId))
        if err != nil {
            log.Fatalf("Failed to create the output file: %v", err)
        }
        defer file.Close()

        // write the data
        for word, files := range wordFiles {
            file.WriteString(fmt.Sprintf("%v %v\n", word, strings.Join(files, " ")))
        }
    }

    // send completion update to the master
    res, err := client.Update(context.Background(), &pb.UpdateRequest{
        WorkerId: task.WorkerId,
        WorkerType: "reducer",
        TaskId: task.TaskId,
        Status: "done",
    })
    if err != nil {
        log.Fatalf("Failed to send completion update: %v", err)
    }

    // check if the task is done
    if res.Response != "Success" {
        log.Fatalf("Failed to send completion update: %v", res.Response)
    } else {
        log.Printf("Task %v done", task.TaskId)
    }
}

// Worker Server
func main() {
    // get the ip address and port of the master
    masterFlag := flag.String("master", "localhost:50051", "Master address")
    typeFlag := flag.String("type", "mapper", "type of the worker (mapper or reducer)")
    flag.Parse()

    // set the master address
    master = *masterFlag

    // check the type of the worker
    if *typeFlag == "mapper" {
        Map()
    } else if *typeFlag == "reducer" {
        Reduce()
    } else {
        log.Fatalf("Invalid worker type: %v", *typeFlag)
    }
}
