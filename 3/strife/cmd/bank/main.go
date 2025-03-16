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
	"syscall"
	"time"
	"io/ioutil"

	gwbs "strife/proto/gw-bank"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
)

const (
	// Key prefix for worker information
	banksPrefix = "/services/banks/"
	leaseTTL    = 10 // 10 seconds
)

type Transaction struct {
	ClientName string `json:"client_name"`
	TransactionType string `json:"transaction_type"`
	Amount float64 `json:"amount"`
	TransactionId string `json:"transaction_id"`
}

type User struct {
	Username string `json:"username"`
	AccountNumber string `json:"account_number"`
	Balance float64 `json:"balance"`
	Transactions []Transaction `json:"transactions"`
}

type BankDatabase struct {
	Bank struct {
		Name string `json:"name"`
		Entries []User `json:"entries"`
		Transactions []Transaction `json:"transactions"`
	} `json:"bank"`
}

// Represents a payment gateway server
type BankServer struct {
	gwbs.UnimplementedBankServiceServer
	id 		string
	address string
	database BankDatabase
}

// RegisterClient implements the RPC method
func (s *BankServer) RegisterClient(ctx context.Context, req *gwbs.ClientDetails) (*gwbs.RegisterResponse, error) {
	log.Printf("RegisterClient called for %s", req.ClientName)

	// Check if the user already exists
	for _, user := range s.database.Bank.Entries {
		if user.Username == req.ClientName {
			return &gwbs.RegisterResponse{
				RegistrationSuccess: true,
			}, nil
		}
	}

	// Add the new user
	newUser := User{
		Username: req.ClientName,
		AccountNumber: fmt.Sprintf("%s-%d", s.id, len(s.database.Bank.Entries)),
		Balance: 0,
	}
	s.database.Bank.Entries = append(s.database.Bank.Entries, newUser)

	// Save the updated database
	data, err := json.Marshal(s.database)
	if err != nil {
		log.Fatalf("Failed to marshal bank database: %v", err)
	}
	log.Printf("Database: %s", data)
	log.Printf("%d", len(s.database.Bank.Entries))

	err = ioutil.WriteFile(fmt.Sprintf("cmd/bank/databases/%s.json", s.id), data, 0644)
	if err != nil {
		log.Printf("Failed to write bank database: %v", err)
		return &gwbs.RegisterResponse{
			RegistrationSuccess: false,
		}, nil
	}

	return &gwbs.RegisterResponse{
		RegistrationSuccess: true,
	}, nil
}

// UpdateClientDetails implements the RPC method
func (s *BankServer) UpdateClientDetails(ctx context.Context, req *gwbs.UpdateDetails) (*gwbs.UpdateResponse, error) {
	log.Printf("UpdateClientDetails called for %s", req.ClientName)

	// Check if the user exists
	var found bool
	for i, user := range s.database.Bank.Entries {
		if user.Username == req.ClientName {
			found = true
			s.database.Bank.Entries[i].Balance = float64(req.BankBalance)
			break
		}
	}
	if !found {
		return &gwbs.UpdateResponse{
			UpdateSuccess: false,
		}, nil
	}

	// Save the updated database
	data, err := json.Marshal(s.database)
	if err != nil {
		log.Fatalf("Failed to marshal bank database: %v", err)
	}

	err = ioutil.WriteFile(fmt.Sprintf("cmd/bank/databases/%s.json", s.id), data, 0644)
	if err != nil {
		log.Fatalf("Failed to write bank database: %v", err)
	}

	return &gwbs.UpdateResponse{
		UpdateSuccess: true,
	}, nil
}

// ViewBalance implements the RPC method
func (s *BankServer) ViewBalance(ctx context.Context, req *gwbs.ViewBalanceRequest) (*gwbs.ViewBalanceResponse, error) {
	log.Printf("ViewBalance called for %s", req.ClientName)

	// Check if the user exists
	for _, user := range s.database.Bank.Entries {
		if user.Username == req.ClientName {
			return &gwbs.ViewBalanceResponse{
				AccountExists: true,
				AccountBalance: float32(user.Balance),
			}, nil
		}
	}

	return &gwbs.ViewBalanceResponse{
		AccountExists: false,
		AccountBalance: float32(0),
	}, nil
}

// InitiateTransaction implements the RPC method
func (s *BankServer) InitiateTransaction(ctx context.Context, req *gwbs.InitiateTransactionRequest) (*gwbs.InitiateTransactionResponse, error) {
	log.Printf("InitiateTransaction called for %s", req.ClientName)

	// Check if the transaction ID already exists
	for _, transaction := range s.database.Bank.Transactions {
		if transaction.TransactionId == req.TransactionId {
			return &gwbs.InitiateTransactionResponse{
				TransactionSuccess: true,
				TransactionMessage: "Transaction ID already exists",
			}, nil
		}
	}

	// Check if the user exists
	var found bool
	for i, user := range s.database.Bank.Entries {
		if user.Username == req.ClientName {
			found = true

			if req.TransactionType == "SEND" {
				// Check if the user has enough balance
				if user.Balance < req.Amount {
					return &gwbs.InitiateTransactionResponse{
						TransactionSuccess: false,
						TransactionMessage: "Insufficient balance",
					}, nil
				}

				// Deduct the amount from the user
				s.database.Bank.Entries[i].Balance -= req.Amount

				// Save the updated database
				data, err := json.Marshal(s.database)
				if err != nil {
					log.Fatalf("Failed to marshal bank database: %v", err)
				}
				err = ioutil.WriteFile(fmt.Sprintf("cmd/bank/databases/%s.json", s.id), data, 0644)
				if err != nil {
					log.Fatalf("Failed to write bank database: %v", err)
				}
			} else if req.TransactionType == "RECEIVE" {
				// Add the amount to the user
				s.database.Bank.Entries[i].Balance += req.Amount

				// Save the updated database
				data, err := json.Marshal(s.database)
				if err != nil {
					log.Fatalf("Failed to marshal bank database: %v", err)
				}
				err = ioutil.WriteFile(fmt.Sprintf("cmd/bank/databases/%s.json", s.id), data, 0644)
				if err != nil {
					log.Fatalf("Failed to write bank database: %v", err)
				}
			} else {
				return &gwbs.InitiateTransactionResponse{
					TransactionSuccess: false,
					TransactionMessage: "Invalid transaction type",
				}, nil
			}
		}
	}

	if !found {
		return &gwbs.InitiateTransactionResponse{
			TransactionSuccess: false,
			TransactionMessage: "User not found",
		}, nil
	}

	// Add the transaction to the database
	transaction := Transaction{
		ClientName: req.ClientName,
		TransactionType: req.TransactionType,
		Amount: req.Amount,
		TransactionId: req.TransactionId,
	}
	s.database.Bank.Transactions = append
	(s.database.Bank.Transactions, transaction)

	// Save the updated database
	data, err := json.Marshal(s.database)
	if err != nil {
		log.Fatalf("Failed to marshal bank database: %v", err)
	}
	err = ioutil.WriteFile(fmt.Sprintf("cmd/bank/databases/%s.json", s.id), data, 0644)
	if err != nil {
		log.Fatalf("Failed to write bank database: %v", err)
	}

	return &gwbs.InitiateTransactionResponse{
		TransactionSuccess: true,
		TransactionMessage: "Transaction successful",
	}, nil
}

// Main function
func main() {
	// Parse command line flags
	port := flag.Int("port", 0, "The server port")
	id := flag.String("id", "bank0", "The bank server ID")
	flag.Parse()

	if *id == "" || *port == 0 {
		log.Fatalf("Bank ID and port number are required. Example: -id=bank1 -port=50052")
	}
	if *port <= 1024 {
		log.Fatalf("Invalid port number: %d", *port)
	}

	address := fmt.Sprintf("localhost:%d", *port)

	// Create a new etcd client
	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatalf("Failed to create etcd client: %v", err)
	}
	defer etcdClient.Close()

	// load the bank database
	data, err := ioutil.ReadFile(fmt.Sprintf("cmd/bank/databases/%s.json", *id))
	if err != nil {
		log.Fatalf("Failed to read bank database: %v", err)
	}

	// Unmarshal the JSON data
	db := BankDatabase{}
	err = json.Unmarshal(data, &db)
	if err != nil {
		log.Fatalf("Failed to unmarshal bank database: %v", err)
	}

	// Create the bank server instance
	server := &BankServer{
		id:      *id,
		address: address,
		database: db,
	}

	// Register with etcd
	ctx := context.Background()
	leaseResp, err := etcdClient.Grant(ctx, leaseTTL)
	if err != nil {
		log.Fatalf("Failed to create lease: %v", err)
	}

	// Create a lease to keep the bank server alive
	_, err = etcdClient.Put(ctx, banksPrefix+server.id, server.address, clientv3.WithLease(leaseResp.ID))
	if err != nil {
		log.Fatalf("Failed to register bank: %v", err)
	}
	log.Printf("Bank registered with etcd: %s at %s", server.id, server.address)

	// Keep the lease alive
	keepAlive, err := etcdClient.KeepAlive(ctx, leaseResp.ID)
	if err != nil {
		log.Fatalf("Failed to keep lease alive: %v", err)
	}

	go func() {
		for range keepAlive {
			// Just drain the channel
		}
	}()

	// Create a new gRPC server
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	Server := grpc.NewServer()
	gwbs.RegisterBankServiceServer(Server, server)

	// Handle graceful shutdown
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig

		// Revoke the lease
		etcdClient.Revoke(ctx, leaseResp.ID)

		// Stop the gRPC server
		log.Println("Shutting down the server...")
		Server.GracefulStop()
	}()

	log.Printf("Bank server %s is running on port %d...", *id, *port)
	if err := Server.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
