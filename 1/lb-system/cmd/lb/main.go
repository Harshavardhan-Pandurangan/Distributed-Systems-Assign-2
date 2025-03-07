package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"
	"flag"

	hbpb "lb-system/proto/heartbeat"
	pb "lb-system/proto/lb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
)

// represents metadata about a worker server
type WorkerInfo struct {
	ID        string 	`json:"id"`
	Address   string 	`json:"address"` // host:port
}

// represents a worker server
type workerServer struct {
	id        	string
	address   	string
	load      	string
	conn 		*grpc.ClientConn
	client 		hbpb.HeartbeatClient
}

// constants for etcd
const (
	// Key prefix for worker information
	workersPrefix = "/services/workers/"
	leaseTTL      = 10 // 10 seconds
)

var ctx = context.Background()
var s_type = ""
var curr_workers []workerServer
var last_assigned_worker int

type loadBalancerServer struct {
	pb.UnimplementedLoadBalancerServer
	etcdClient *clientv3.Client
}

func updateWorkers(etcdClient *clientv3.Client) {
	// get all the workers from etcd
	ctx := context.Background()
	resp, err := etcdClient.Get(ctx, workersPrefix, clientv3.WithPrefix())
	if err != nil {
		log.Printf("Failed to get workers: %v", err)
		return
	}

	// make this a list of workerIds, addresses and loads
	workers := make([]WorkerInfo, 0)
	for _, kv := range resp.Kvs {
		var worker WorkerInfo
		if err := json.Unmarshal(kv.Value, &worker); err != nil {
			log.Printf("Failed to unmarshal worker info: %v", err)
			continue
		}
		workers = append(workers, worker)
	}

	// update the current workers
	for _, worker := range workers {
		found := false
		for _, curr_worker := range curr_workers {
			if worker.ID == curr_worker.id {
				found = true
				break
			}
		}
		if !found {
			// curr_workers = append(curr_workers, worker)
			curr_workers = append(curr_workers, workerServer{
				id: worker.ID,
				address: worker.Address,
				load: "",
				conn: nil,
				client: nil,
			})
			// make a gRPC connection to the worker
			conn, err := grpc.Dial(worker.Address, grpc.WithInsecure())
			if err != nil {
				log.Fatalf("Failed to connect to worker: %v", err)
			}
			curr_workers[len(curr_workers)-1].conn = conn
			curr_workers[len(curr_workers)-1].client = hbpb.NewHeartbeatClient(conn)
		}
	}

	// remove current workers that are not in the workers list in place
	for i := 0; i < len(curr_workers); i++ {
		found := false
		for _, worker := range workers {
			if worker.ID == curr_workers[i].id {
				found = true
				break
			}
		}
		if !found {
			// remove the worker
			curr_workers[i].conn.Close()
			curr_workers = append(curr_workers[:i], curr_workers[i+1:]...)
		}
	}

	// log the current workers
	log.Printf("Current workers: %v", curr_workers)

	// update the load of the workers
	for i := 0; i < len(curr_workers); i++ {
		// get the worker's load
		workerClient := curr_workers[i].client
		workerResp, err := workerClient.SendHeartbeat(ctx, &hbpb.HeartbeatRequest{
			WorkerId: curr_workers[i].id,
		})
		if err != nil {
			log.Fatalf("Failed to send heartbeat to worker: %v", err)
		}
		curr_workers[i].load = workerResp.WorkerLoad

		// log the worker's load
		log.Printf("Worker %s at %s has load %s", workerResp.WorkerId, workerResp.WorkerAddress, workerResp.WorkerLoad)
	}
}

// // finds an available worker with the least load
// func (s *loadBalancerServer) getAvailableWorker() (WorkerInfo, error) {
// 	ctx := context.Background()

// 	// Get all worker registrations
// 	resp, err := s.etcdClient.Get(ctx, workersPrefix, clientv3.WithPrefix())
// 	if err != nil {
// 		return WorkerInfo{}, fmt.Errorf("failed to get workers: %v", err)
// 	}

// 	if len(resp.Kvs) == 0 {
// 		return WorkerInfo{}, fmt.Errorf("no workers available")
// 	}

// 	var bestWorker WorkerInfo
// 	bestLoad := -1.0

// 	// Find the worker with the least load
// 	for _, kv := range resp.Kvs {
// 		var worker WorkerInfo
// 		if err := json.Unmarshal(kv.Value, &worker); err != nil {
// 			log.Printf("Failed to unmarshal worker info: %v", err)
// 			continue
// 		}

// 		// Select worker with the least load
// 		if bestLoad == -1 || worker.Load < bestLoad {
// 			bestWorker = worker
// 			bestLoad = worker.Load
// 		}
// 	}

// 	if bestLoad == -1 {
// 		return WorkerInfo{}, fmt.Errorf("no available workers found")
// 	}

// 	return bestWorker, nil
// }

func (s *loadBalancerServer) getNextWorker() (workerServer, error) {
	if len(curr_workers) == 0 {
		return workerServer{}, fmt.Errorf("no workers available")
	}

	if s_type == "first" {
		// get the first worker
		worker := curr_workers[0]
		curr_workers = append(curr_workers[1:], worker)
		return worker, nil
	} else if s_type == "rr" {
		// get the next worker in the list
		last_assigned_worker = (last_assigned_worker + 1) % len(curr_workers)
		worker := curr_workers[last_assigned_worker]
		return worker, nil
	} else if s_type == "ll" {
		// get the worker with the least load
		bestWorker := curr_workers[0]
		bestLoad := bestWorker.load
		bestIndex := 0
		for i := 1; i < len(curr_workers); i++ {
			if curr_workers[i].load < bestLoad {
				bestWorker = curr_workers[i]
				bestLoad = curr_workers[i].load
				bestIndex = i
			}
		}
		return curr_workers[bestIndex], nil
	} else {
		return workerServer{}, fmt.Errorf("invalid scheduling type")
	}
}

// implements the gRPC method for client requests
func (s *loadBalancerServer) GetWorker(ctx context.Context, req *pb.WorkerRequest) (*pb.WorkerResponse, error) {
	// worker, err := s.getAvailableWorker()
	worker, err := s.getNextWorker()
	if err != nil {
		return &pb.WorkerResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	// Log the assignment
	log.Printf("Assigning client %s to worker %s (%s)", req.ClientId, worker.id, worker.address)

	return &pb.WorkerResponse{
		WorkerId:     worker.id,
		WorkerAddress: worker.address,
		Success:      true,
	}, nil
}

// main function
func main() {
	// get the flag for the scheduling type
	flag.StringVar(&s_type, "type", "rr", "Scheduling type")

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

	// // Watch for worker changes (for logging/monitoring)
	// watchCh := etcdClient.Watch(context.Background(), workersPrefix, clientv3.WithPrefix())
	// go func() {
	// 	for watchResp := range watchCh {
	// 		for _, event := range watchResp.Events {
	// 			log.Printf("Worker change detected: %q",
	// 				event.Kv.Key)
	// 		}
	// 	}
	// }()

	// routine to maintain worker status (heartbeat)
	go func() {
		for {
			updateWorkers(etcdClient)
			time.Sleep(2 * time.Second)
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
