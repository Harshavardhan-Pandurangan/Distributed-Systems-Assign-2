// package main

// import (
//     "context"
//     // "fmt"
//     "log"
//     "net"
//     "sync"

//     pb "mapreduce/proto/mapreduce"

//     "google.golang.org/grpc"
// )

// type server struct {
//     pb.UnimplementedMapReduceServiceServer
//     mu       sync.Mutex
//     tasks    []pb.Task
//     taskIdx  int
// }

// func (s *server) RequestTask(ctx context.Context, req *pb.WorkerRequest) (*pb.Task, error) {
//     s.mu.Lock()
//     defer s.mu.Unlock()

//     if s.taskIdx >= len(s.tasks) {
//         return &pb.Task{TaskType: "none"}, nil
//     }

//     task := s.tasks[s.taskIdx]
//     s.taskIdx++
//     log.Printf("Assigned %s task: %v to Worker %s", task.TaskType, task.TaskId, req.WorkerId)
//     return &task, nil
// }

// func (s *server) SubmitTask(ctx context.Context, res *pb.TaskResult) (*pb.Ack, error) {
//     log.Printf("Worker completed task: %d of type %s. Output: %s", res.TaskId, res.TaskType, res.ResultFile)
//     return &pb.Ack{Success: true}, nil
// }

// func main() {
//     lis, err := net.Listen("tcp", ":50051")
//     if err != nil {
//         log.Fatalf("Failed to listen: %v", err)
//     }

//     grpcServer := grpc.NewServer()
//     pb.RegisterMapReduceServiceServer(grpcServer, &server{
//         tasks: []pb.Task{
//             {TaskType: "map", FileName: "input1.txt", TaskId: 1, NumReduce: 2},
//             {TaskType: "map", FileName: "input2.txt", TaskId: 2, NumReduce: 2},
//             {TaskType: "reduce", FileName: "intermediate", TaskId: 3, NumReduce: 0},
//         },
//     })

//     log.Println("Master server listening on port 50051...")
//     if err := grpcServer.Serve(lis); err != nil {
//         log.Fatalf("Failed to serve: %v", err)
//     }
// }


package main

import (
    "context"
    // "fmt"
    "log"
    "net"
    "os"
    // "strings"
    "sync"

    pb "mapreduce/proto/mapreduce"

    "google.golang.org/grpc"
)

type server struct {
    pb.UnimplementedMapReduceServiceServer
    mu      sync.Mutex
    tasks   []pb.Task
    taskIdx int
}

func (s *server) RequestTask(ctx context.Context, req *pb.WorkerRequest) (*pb.Task, error) {
    s.mu.Lock()
    defer s.mu.Unlock()

    if s.taskIdx >= len(s.tasks) {
        return &pb.Task{TaskType: "none"}, nil
    }

    task := s.tasks[s.taskIdx]
    s.taskIdx++
    log.Printf("Assigned %s task: %d to Worker %s", task.TaskType, task.TaskId, req.WorkerId)
    return &task, nil
}

func (s *server) SubmitTask(ctx context.Context, res *pb.TaskResult) (*pb.Ack, error) {
    log.Printf("Worker completed task: %d of type %s. Output: %s", res.TaskId, res.TaskType, res.ResultFile)
    return &pb.Ack{Success: true}, nil
}

func main() {
    lis, err := net.Listen("tcp", ":50051")
    if err != nil {
        log.Fatalf("Failed to listen: %v", err)
    }

    files, _ := os.ReadDir("../../files/input")

    var tasks []pb.Task
    for i, file := range files {
        tasks = append(tasks, pb.Task{TaskType: "map", FileName: file.Name(), TaskId: int32(i + 1), NumReduce: 2})
    }
    tasks = append(tasks, pb.Task{TaskType: "reduce", FileName: "intermediate", TaskId: int32(len(files) + 1), NumReduce: 0})

    grpcServer := grpc.NewServer()
    pb.RegisterMapReduceServiceServer(grpcServer, &server{tasks: tasks})

    log.Println("Master server listening on port 50051...")
    if err := grpcServer.Serve(lis); err != nil {
        log.Fatalf("Failed to serve: %v", err)
    }
}
