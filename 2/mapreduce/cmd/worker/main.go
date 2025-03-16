// package main

// import (
//     "context"
//     "fmt"
//     "log"
//     "os"
//     "time"

//     pb "mapreduce/proto/mapreduce"

//     "google.golang.org/grpc"
// )

// func processMapTask(task *pb.Task) string {
//     outputFile := fmt.Sprintf("map_output_%d.txt", task.TaskId)
//     f, _ := os.Create(outputFile)
//     defer f.Close()
//     fmt.Fprintf(f, "Processed map task %d\n", task.TaskId)
//     return outputFile
// }

// func processReduceTask(task *pb.Task) string {
//     outputFile := fmt.Sprintf("reduce_output_%d.txt", task.TaskId)
//     f, _ := os.Create(outputFile)
//     defer f.Close()
//     fmt.Fprintf(f, "Processed reduce task %d\n", task.TaskId)
//     return outputFile
// }

// func main() {
//     conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
//     if err != nil {
//         log.Fatalf("Failed to connect to master: %v", err)
//     }
//     defer conn.Close()

//     client := pb.NewMapReduceServiceClient(conn)

//     for {
//         task, err := client.RequestTask(context.Background(), &pb.WorkerRequest{WorkerId: "worker1"})
//         if err != nil {
//             log.Fatalf("Failed to request task: %v", err)
//         }
//         if task.TaskType == "none" {
//             log.Println("No tasks available. Exiting.")
//             break
//         }

//         var resultFile string
//         if task.TaskType == "map" {
//             resultFile = processMapTask(task)
//         } else if task.TaskType == "reduce" {
//             resultFile = processReduceTask(task)
//         }

//         _, err = client.SubmitTask(context.Background(), &pb.TaskResult{
//             TaskId:     task.TaskId,
//             TaskType:   task.TaskType,
//             ResultFile: resultFile,
//         })
//         if err != nil {
//             log.Fatalf("Failed to submit task: %v", err)
//         }

//         time.Sleep(2 * time.Second)
//     }
// }


package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "strings"
    "time"

    pb "mapreduce/proto/mapreduce"

    "google.golang.org/grpc"
)

// Map function: Reads a file, counts words, and saves intermediate output
func processMapTask(task *pb.Task) string {
    file, err := os.ReadFile("../datasets/" + task.FileName)
    if err != nil {
        log.Fatalf("Failed to read file: %v", err)
    }

    words := strings.Fields(string(file))
    wordCounts := make(map[string]int)
    for _, word := range words {
        wordCounts[word]++
    }

    outputFile := fmt.Sprintf("../output/map_output_%d.txt", task.TaskId)
    f, _ := os.Create(outputFile)
    defer f.Close()

    for word, count := range wordCounts {
        fmt.Fprintf(f, "%s %d\n", word, count)
    }
    return outputFile
}

// Reduce function: Aggregates word counts from intermediate files
func processReduceTask(task *pb.Task) string {
    intermediateFiles, _ := os.ReadDir("../output")

    wordCounts := make(map[string]int)
    for _, file := range intermediateFiles {
        content, _ := os.ReadFile("../output/" + file.Name())
        lines := strings.Split(string(content), "\n")
        for _, line := range lines {
            if line == "" {
                continue
            }
            var word string
            var count int
            fmt.Sscanf(line, "%s %d", &word, &count)
            wordCounts[word] += count
        }
    }

    outputFile := "../output/final_output.txt"
    f, _ := os.Create(outputFile)
    defer f.Close()

    for word, count := range wordCounts {
        fmt.Fprintf(f, "%s %d\n", word, count)
    }
    return outputFile
}

func main() {
    conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
    if err != nil {
        log.Fatalf("Failed to connect to master: %v", err)
    }
    defer conn.Close()

    client := pb.NewMapReduceServiceClient(conn)

    for {
        task, err := client.RequestTask(context.Background(), &pb.WorkerRequest{WorkerId: "worker1"})
        if err != nil {
            log.Fatalf("Failed to request task: %v", err)
        }
        if task.TaskType == "none" {
            log.Println("No tasks available. Exiting.")
            break
        }

        var resultFile string
        if task.TaskType == "map" {
            resultFile = processMapTask(task)
        } else if task.TaskType == "reduce" {
            resultFile = processReduceTask(task)
        }

        _, err = client.SubmitTask(context.Background(), &pb.TaskResult{
            TaskId:     task.TaskId,
            TaskType:   task.TaskType,
            ResultFile: resultFile,
        })
        if err != nil {
            log.Fatalf("Failed to submit task: %v", err)
        }

        time.Sleep(2 * time.Second)
    }
}
