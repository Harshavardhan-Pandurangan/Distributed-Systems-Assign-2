package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"

	pb "lb-system/proto/lb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
)

// represents metadata about a worker server
type WorkerInfo struct {
	ID        string `json:"id"`
	Address   string `json:"address"` // host:port
	Load      int    `json:"load"`    // Current load/connections
	MaxLoad   int    `json:"maxLoad"` // Maximum capacity
	Available bool   `json:"available"`
}

// constants for etcd
const (
	// Key prefix for worker information
	workersPrefix = "/services/workers/"
	leaseTTL      = 10 // 10 seconds
)

type loadBalancerServer struct {
	pb.UnimplementedLoadBalancerServer
	etcdClient *clientv3.Client
}

// finds an available worker with the least load
func (s *loadBalancerServer) getAvailableWorker() (WorkerInfo, error) {
	ctx := context.Background()

	// Get all worker registrations
	resp, err := s.etcdClient.Get(ctx, workersPrefix, clientv3.WithPrefix())
	if err != nil {
		return WorkerInfo{}, fmt.Errorf("failed to get workers: %v", err)
	}

	if len(resp.Kvs) == 0 {
		return WorkerInfo{}, fmt.Errorf("no workers available")
	}

	var bestWorker WorkerInfo
	bestLoad := -1

	// Find the worker with the least load
	for _, kv := range resp.Kvs {
		var worker WorkerInfo
		if err := json.Unmarshal(kv.Value, &worker); err != nil {
			log.Printf("Failed to unmarshal worker info: %v", err)
			continue
		}

		// Skip unavailable workers
		if !worker.Available {
			continue
		}

		// Select worker with the least load
		if bestLoad == -1 || worker.Load < bestLoad {
			bestWorker = worker
			bestLoad = worker.Load
		}
	}

	if bestLoad == -1 {
		return WorkerInfo{}, fmt.Errorf("no available workers found")
	}

	return bestWorker, nil
}

// implements the gRPC method for client requests
func (s *loadBalancerServer) GetWorker(ctx context.Context, req *pb.WorkerRequest) (*pb.WorkerResponse, error) {
	worker, err := s.getAvailableWorker()
	if err != nil {
		return &pb.WorkerResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	// Log the assignment
	log.Printf("Assigning client %s to worker %s (%s)", req.ClientId, worker.ID, worker.Address)

	return &pb.WorkerResponse{
		WorkerId:     worker.ID,
		WorkerAddress: worker.Address,
		Success:      true,
	}, nil
}

// main function
func main() {
	// Connect to etcd
	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatalf("Failed to connect to etcd: %v", err)
	}
	defer etcdClient.Close()

	// Create gRPC server
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Create the load balancer server
	lbServer := &loadBalancerServer{
		etcdClient: etcdClient,
	}

	// Watch for worker changes (for logging/monitoring)
	watchCh := etcdClient.Watch(context.Background(), workersPrefix, clientv3.WithPrefix())
	go func() {
		for watchResp := range watchCh {
			for _, event := range watchResp.Events {
				log.Printf("Worker change detected: %s %q",
					string(event.Type), event.Kv.Key)
			}
		}
	}()

	// Start gRPC server
	grpcServer := grpc.NewServer()
	pb.RegisterLoadBalancerServer(grpcServer, lbServer)

	log.Printf("Load Balancer server listening on :50051")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
