package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	wpb "lb-system/proto/worker"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
)

// represents metadata about a worker server
type WorkerInfo struct {
	ID        string `json:"id"`
	Address   string `json:"address"`
	Load      int    `json:"load"`
	MaxLoad   int    `json:"maxLoad"`
	Available bool   `json:"available"`
}

// constants for etcd
const (
	// Key prefix for worker information
	workersPrefix = "/services/workers/"
	leaseTTL      = 10 // 10 seconds
)

// represents a worker server
type workerServer struct {
	wpb.UnimplementedWorkerServiceServer
	id        string
	address   string
	load      int32 // Using atomic for concurrent access
	maxLoad   int
	available bool
}

// implements the worker service method
func (s *workerServer) DoWork(ctx context.Context, req *wpb.WorkRequest) (*wpb.WorkResponse, error) {
	// Increment load counter when work starts
	atomic.AddInt32(&s.load, 1)
	defer atomic.AddInt32(&s.load, -1) // Decrement when work is done

	log.Printf("Processing work request: %s", req.TaskData)

	// // Simulate actual work
	// time.Sleep(2 * time.Second)

	// check the type of task
	if len(req.TaskData) > 5 && req.TaskData[:5] == "sleep" {
		// if task starts with "sleep", followed by a number, sleep for that duration
		sleepDuration, err := time.ParseDuration(req.TaskData[6:])
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
	} else if len(req.TaskData) > 5 && req.TaskData[:5] == "sum-up" {
		// if task starts with "sum-up", followed by a number, sum up all numbers from 1 to that number
		n, err := fmt.Sscanf(req.TaskData[6:], "%d")
		if err != nil || n != 1 {
			return &wpb.WorkResponse{
				Success: false,
				Result:  fmt.Sprintf("Invalid number: %s", req.TaskData[6:]),
			}, nil
		}

		sum := 0
		for i := 1; i <= n; i++ {
			sum += i
		}

		return &wpb.WorkResponse{
			Success: true,
			Result:  fmt.Sprintf("Sum of numbers from 1 to %d is %d", n, sum),
		}, nil
	}

	// // Return result
	// return &wpb.WorkResponse{
	// 	Success: true,
	// 	Result:  fmt.Sprintf("Processed: %s by worker %s", req.TaskData, s.id),
	// }, nil

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
		load:      0,
		maxLoad:   100,
		available: true,
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
		Load:      int(worker.load),
		MaxLoad:   worker.maxLoad,
		Available: worker.available,
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

	// Start a goroutine to update worker info
	go func() {
		for {
			// Update with current load
			workerInfo.Load = int(atomic.LoadInt32(&worker.load))
			workerJSON, _ := json.Marshal(workerInfo)

			_, err := etcdClient.Put(ctx, key, string(workerJSON), clientv3.WithLease(leaseResp.ID))
			if err != nil {
				log.Printf("Failed to update worker info: %v", err)
			}

			time.Sleep(5 * time.Second)
		}
	}()

	// Start gRPC server
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	wpb.RegisterWorkerServiceServer(grpcServer, worker)

	// Handle graceful shutdown
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c

		// Mark worker as unavailable
		workerInfo.Available = false
		workerJSON, _ := json.Marshal(workerInfo)
		etcdClient.Put(ctx, key, string(workerJSON), clientv3.WithLease(leaseResp.ID))

		log.Println("Shutting down worker server...")
		grpcServer.GracefulStop()
	}()

	log.Printf("Worker server %s listening on %s", worker.id, worker.address)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
