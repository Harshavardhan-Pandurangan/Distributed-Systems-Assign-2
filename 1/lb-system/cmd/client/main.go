package main

import (
	"context"
	"flag"
	"log"
	"time"
	"fmt"

	lbpb "lb-system/proto/lb"
	wpb "lb-system/proto/worker"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// main function
func main() {
	// Parse command line flags
	clientID := flag.String("id", "client1", "Client ID")
	lbAddress := flag.String("lb", "localhost:50051", "Load balancer address")
	task := flag.String("task", "default task", "Task to send to worker")
	flag.Parse()

	// print the task
	log.Printf("Sending task to worker: %s", *task)

	current_time := time.Now()

	// Connect to load balancer
	log.Printf("Connecting to load balancer at %s", *lbAddress)
	lbConn, err := grpc.Dial(*lbAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to load balancer: %v", err)
	}
	defer lbConn.Close()

	// Create load balancer client
	lbClient := lbpb.NewLoadBalancerClient(lbConn)

	// Request a worker
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	workerResp, err := lbClient.GetWorker(ctx, &lbpb.WorkerRequest{
		ClientId: *clientID,
	})
	if err != nil {
		log.Fatalf("Failed to get worker: %v", err)
	}

	if !workerResp.Success {
		log.Fatalf("Load balancer error: %s", workerResp.ErrorMessage)
	}

	log.Printf("Received worker assignment: %s at %s",
		workerResp.WorkerId, workerResp.WorkerAddress)

	// Connect to the assigned worker
	workerConn, err := grpc.Dial(workerResp.WorkerAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to worker: %v", err)
	}
	defer workerConn.Close()

	// Create worker client
	workerClient := wpb.NewWorkerServiceClient(workerConn)

	// Send work request
	// workCtx, workCancel := context.WithTimeout(context.Background(), 10*time.Second)
	// without timeout
	workCtx, workCancel := context.WithCancel(context.Background())
	defer workCancel()

	workResp, err := workerClient.DoWork(workCtx, &wpb.WorkRequest{
		TaskData: *task,
	})
	if err != nil {
		log.Fatalf("Work request failed: %v", err)
	}

	log.Printf("Work completed successfully: %s", workResp.Result)

	fmt.Printf("Time taken: %s", time.Since(current_time))
}
