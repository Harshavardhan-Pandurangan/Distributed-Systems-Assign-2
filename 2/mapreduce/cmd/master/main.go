package main

import (
    "context"
    "flag"
    "log"
    "net"
    "sync"
    "io/ioutil"
    "fmt"
    "os"
    "time"

    pb "mapreduce/proto/mapreduce"
    "google.golang.org/grpc"
)

type masterServer struct {
    pb.UnimplementedMapReduceServiceServer
    numReducers int
    inputFolder string
    fileNames []string
    task int32
    mu sync.Mutex
    mappers map[int32]string
    reducers map[int32]string
}

// Mapper function
func (s *masterServer) Map(ctx context.Context, req *pb.MapRequest) (*pb.MapResponse, error) {
    // lock the mutex
    s.mu.Lock()
    defer s.mu.Unlock()

    // assign worker id
    workerID := len(s.mappers)
    if workerID >= len(s.fileNames) {
        return &pb.MapResponse{
            WorkerId: -2,
            InputFile: "",
            NumReduceTasks: 0,
            TaskId: 0,
        }, nil
    }
    s.mappers[int32(workerID)] = s.fileNames[workerID]

    // return the response
    return &pb.MapResponse{
        WorkerId: int32(workerID),
        InputFile: s.inputFolder + "/" + s.fileNames[workerID],
        NumReduceTasks: int32(s.numReducers),
        TaskId: s.task,
    }, nil
}

// Reducer Function
func (s *masterServer) Reduce(ctx context.Context, req *pb.ReduceRequest) (*pb.ReduceResponse, error) {
    // lock the mutex
    s.mu.Lock()
    defer s.mu.Unlock()

    // check if all mappers are done
    if len(s.mappers) == len(s.fileNames) {
        for _, value := range s.mappers {
            if value != "done" {
                return &pb.ReduceResponse{
                    WorkerId: -2,
                    IntermediateFiles: "",
                    TaskId: 0,
                }, nil
            }
        }
    } else {
        return &pb.ReduceResponse{
            WorkerId: -2,
            IntermediateFiles: "",
            TaskId: 0,
        }, nil
    }

    // assign worker id
    workerID := len(s.reducers)
    s.reducers[int32(workerID)] = "datasets/intermediate"

    // return the response
    return &pb.ReduceResponse{
        WorkerId: int32(workerID),
        IntermediateFiles: "datasets/intermediate",
        TaskId: s.task,
    }, nil
}

// Work Update Function
func (s *masterServer) Update(ctx context.Context, req *pb.UpdateRequest) (*pb.UpdateResponse, error) {
    // lock the mutex
    s.mu.Lock()
    defer s.mu.Unlock()

    // check if the worker is a mapper or a reducer
    if req.WorkerType == "mapper" {
        // mapper
        s.mappers[req.WorkerId] = "done"
    } else {
        // reducer
        s.reducers[req.WorkerId] = "done"
    }

    // return the response
    return &pb.UpdateResponse{
        Response: "Success",
    }, nil
}

// Master server
func main() {
    // Parse command line arguments
    numReducers := flag.Int("reducers", 3, "Number of reducers")
    task := flag.String("task", "wordcount", "Task to perform")
    inputFolder := flag.String("input", "datasets/input", "Folder containing input files")
    flag.Parse()

    // Go through the input folder and count the number of files
    files, err := ioutil.ReadDir(*inputFolder)
    if err != nil {
        log.Fatalf("Failed to read input folder: %v", err)
    }
    for i := 0; i < len(files); i++ {
        if files[i].IsDir() {
            files = append(files[:i], files[i+1:]...)
            i--
        }
    }
    fmt.Printf("Number of input files: %d\n", len(files))

    // check if "datasets" folder exists, if not create it
    if _, err := os.Stat("datasets"); os.IsNotExist(err) {
        os.Mkdir("datasets", 0755)
    }
    if _, err := os.Stat("datasets/intermediate"); os.IsNotExist(err) {
        os.Mkdir("datasets/intermediate", 0755)
    }
    if _, err := os.Stat("datasets/output"); os.IsNotExist(err) {
        os.Mkdir("datasets/output", 0755)
    }

    // Create a new gRPC server
    lis, err := net.Listen("tcp", ":50051")
    if err != nil {
        log.Fatalf("Failed to listen: %v", err)
    }
    task_ := 0
    if *task == "wordcount" {
        task_ = 1
    } else if *task == "invertedindex" {
        task_ = 2
    }
    s := grpc.NewServer()
    files_ := make([]string, len(files))
    for i, file := range files {
        files_[i] = file.Name()
    }
    var MasterServer masterServer = masterServer{
        numReducers: *numReducers,
        inputFolder: *inputFolder,
        fileNames: files_,
        task: int32(task_),
        mappers: make(map[int32]string),
        reducers: make(map[int32]string),
    }

    pb.RegisterMapReduceServiceServer(s, &MasterServer)

    go func() {
        for {
            log.Printf("Checking if all mappers and reducers are done")
            // check if all mappers are done
            var done bool = true
            if len(MasterServer.mappers) != len(files) {
                done = false
            }
            for _, value := range MasterServer.mappers {
                log.Printf("Mapper: %s", value)
                if value != "done" {
                    done = false
                    break
                }
            }

            // check if all reducers are done
            var done_ bool = true
            if len(MasterServer.reducers) != *numReducers {
                done_ = false
            }
            for _, value := range MasterServer.reducers {
                log.Printf("Reducer: %s", value)
                if value != "done" {
                    done_ = false
                    break
                }
            }

            if done && done_ {
                // all mappers and reducers are done
                break
            }

            // sleep for 3 second
            time.Sleep(3 * time.Second)
        }

        log.Printf("All mappers and reducers are done")

        // close the server
        s.Stop()

        // exit the program
        os.Exit(0)
    }()

    // Register the server with gRPC
    if err := s.Serve(lis); err != nil {
        log.Fatalf("Failed to serve: %v", err)
    }
}
