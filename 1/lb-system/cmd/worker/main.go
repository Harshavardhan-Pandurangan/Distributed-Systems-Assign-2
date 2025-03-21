package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"strconv"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	wpb "lb-system/proto/worker"
	hbpb "lb-system/proto/heartbeat"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"github.com/shirou/gopsutil/process"
)

// represents metadata about a worker server
type WorkerInfo struct {
	ID        string 	`json:"id"`
	Address   string 	`json:"address"`
}

// constants for etcd
const (
	// Key prefix for worker information
	workersPrefix = "/services/workers/"
	leaseTTL      = 10 // 10 seconds
)

// declare proc as a global variable
var proc *process.Process

// represents a worker server
type workerServer struct {
	wpb.UnimplementedWorkerServiceServer
	hbpb.UnimplementedHeartbeatServer
	id        string
	address   string
	load      string
}

func (s *workerServer) SendHeartbeat(ctx context.Context, req *hbpb.HeartbeatRequest) (*hbpb.HeartbeatStatus, error) {
	// get all info to send with heartbeat
	worker_id := s.id
	worker_address := s.address
	// worker_load, err := proc.CPUPercent()
	// worker_load, err := proc.CPUPercent(time.Second)
	worker_load, err := proc.CPUPercent()

	if err != nil {
		log.Fatalf("Failed to get process: %v", err)

		// send heartbeat status
		return &hbpb.HeartbeatStatus{
			WorkerId: worker_id,
			WorkerAddress: worker_address,
			WorkerLoad: fmt.Sprintf("%f", worker_load),
			ErrorMessage: err.Error(),
		}, err
	}

	// send heartbeat status
	return &hbpb.HeartbeatStatus{
		WorkerId: worker_id,
		WorkerAddress: worker_address,
		// WorkerLoad: worker_load,
		WorkerLoad: fmt.Sprintf("%f", worker_load),
		ErrorMessage: fmt.Sprintf("Worker %s at %s", worker_id, worker_address),
	}, nil
}

// implements the worker service method
func (s *workerServer) DoWork(ctx context.Context, req *wpb.WorkRequest) (*wpb.WorkResponse, error) {
	log.Printf("Processing work request: %s", req.TaskData)

	// check the type of task
	if len(req.TaskData) > 5 && req.TaskData[:5] == "sleep" {
		// if task starts with "sleep", followed by a number, sleep for that duration
		log.Printf("Sleeping for %s", req.TaskData[6:])
		// the sleep time comes in as a string of an integer, convert it to time
		sleepDuration, err := time.ParseDuration(req.TaskData[6:] + "s")
		if err != nil {
			return &wpb.WorkResponse{
				Success: false,
				Result:  fmt.Sprintf("Invalid sleep duration: %s", req.TaskData[6:]),
			}, nil
		}

		time.Sleep(sleepDuration)

		return &wpb.WorkResponse{
			Success: true,
			Result:  fmt.Sprintf("Slept for %s", sleepDuration),
		}, nil
	} else if len(req.TaskData) > 5 && req.TaskData[:6] == "sum-up" {
		// if task starts with "sum-up", followed by a number, sum up all numbers from 1 to that number
		n, err := strconv.Atoi(req.TaskData[7:])
		if err != nil {
			return &wpb.WorkResponse{
				Success: false,
				Result:  fmt.Sprintf("Invalid number: %s", req.TaskData[6:]),
			}, nil
		}

		sum := 0
		for i := 1; i <= n; i++ {
			sum = (sum + i) % 1000000000000
		}

		return &wpb.WorkResponse{
			Success: true,
			Result:  fmt.Sprintf("Sum of numbers from 1 to %d is %d", n, sum),
		}, nil
	}

	// if task is not recognized, return an error
	return &wpb.WorkResponse{
		Success: false,
		Result:  fmt.Sprintf("Unknown task: %s", req.TaskData),
	}, nil
}

// main function
func main() {
	// Parse command line flags
	id := flag.String("id", "", "Worker ID")
	port := flag.Int("port", 0, "Worker port")
	flag.Parse()

	if *id == "" || *port == 0 {
		log.Fatalf("Worker ID and port are required. Example: -id=worker1 -port=50052")
	}

	address := fmt.Sprintf("localhost:%d", *port)

	// Connect to etcd
	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatalf("Failed to connect to etcd: %v", err)
	}
	defer etcdClient.Close()

	// Create the worker server instance
	worker := &workerServer{
		id:        *id,
		address:   address,
		load:      "0",
	}

	// Register with etcd
	ctx := context.Background()
	leaseResp, err := etcdClient.Grant(ctx, leaseTTL)
	if err != nil {
		log.Fatalf("Failed to create lease: %v", err)
	}

	// Prepare worker info for registration
	workerInfo := WorkerInfo{
		ID:        worker.id,
		Address:   worker.address,
	}

	// Convert to JSON
	workerJSON, err := json.Marshal(workerInfo)
	if err != nil {
		log.Fatalf("Failed to marshal worker info: %v", err)
	}

	// Register with etcd
	key := workersPrefix + worker.id
	_, err = etcdClient.Put(ctx, key, string(workerJSON), clientv3.WithLease(leaseResp.ID))
	if err != nil {
		log.Fatalf("Failed to register worker: %v", err)
	}

	log.Printf("Worker registered with etcd: %s at %s", worker.id, worker.address)

	// Keep lease alive
	keepAliveCh, err := etcdClient.KeepAlive(ctx, leaseResp.ID)
	if err != nil {
		log.Fatalf("Failed to keep lease alive: %v", err)
	}

	// Handle lease keep-alive
	go func() {
		for range keepAliveCh {
			// Just drain the channel
		}
	}()

	// let's make proc a global variable
	proc, err = process.NewProcess(int32(os.Getpid()))
	if err != nil {
		log.Fatalf("Failed to get process: %v", err)
	}

	// Start gRPC server
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	hbpb.RegisterHeartbeatServer(grpcServer, worker)
	wpb.RegisterWorkerServiceServer(grpcServer, worker)

	// Handle graceful shutdown
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c

		log.Println("Shutting down worker server...")
		grpcServer.GracefulStop()
	}()

	log.Printf("Worker server %s listening on %s", worker.id, worker.address)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
