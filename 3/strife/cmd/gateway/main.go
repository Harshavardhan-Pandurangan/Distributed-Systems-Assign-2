package main

import (
	"context"
	"log"
	"net"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"crypto/sha256"
	"encoding/json"
	"encoding/hex"
	"strings"
	"time"
	"fmt"
	"strconv"

	clgw "strife/proto/cl-gw"
	gwbs "strife/proto/gw-bank"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
)

type ServerDatabase struct {
	Server struct {
		Name string `json:"name"`
		Entries []User `json:"entries"`
		Transactions []Transaction `json:"transactions"`
	} `json:"server"`
}

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Transaction struct {
	ClientName string `json:"client_name"`
	BankName string `json:"bank_name"`
	Amount float32 `json:"amount"`
	ClientName2 string `json:"client_name2"`
	BankName2 string `json:"bank_name2"`
	TransactionId string `json:"transaction_id"`
}

// list of users
var db ServerDatabase

// constants for etcd
const (
	// Key prefix for worker information
	banksPrefix = "/services/banks/"
	leaseTTL      = 10 // 10 seconds
)

// Represents a payment gateway server
type paymentGatewayServer struct {
	clgw.UnimplementedPaymentGatewayServiceServer
	etcdClient *clientv3.Client
}

// RegisterClient implements the RPC method
func (s *paymentGatewayServer) RegisterClient(ctx context.Context, req *clgw.ClientDetails) (*clgw.RegisterResponse, error) {
	log.Printf("RegisterClient called for %s", req.ClientName)

	// Check if the user already exists
	for _, user := range db.Server.Entries {
		if user.Username == req.ClientName {
			return &clgw.RegisterResponse{
				RegistrationSuccess: false,
			}, nil
		}
	}

	// Hash the password
	encryptedPassword := sha256.Sum256([]byte(req.Password))
	hashedPassword := hex.EncodeToString(encryptedPassword[:])

	db.Server.Entries = append(db.Server.Entries, User{
		Username: req.ClientName,
		Password: hashedPassword,
	})

	// update the server database
	data, err := json.Marshal(db)
	if err != nil {
		log.Fatalf("Failed to marshal server database: %v", err)
	}
	if err := ioutil.WriteFile("cmd/gateway/server.json", data, 0644); err != nil {
		log.Fatalf("Failed to write server database: %v", err)
	}

	return &clgw.RegisterResponse{
		RegistrationSuccess: true,
	}, nil
}

// UpdateClientDetails implements the RPC method
func (s *paymentGatewayServer) UpdateClientDetails(ctx context.Context, req *clgw.UpdateDetails) (*clgw.UpdateResponse, error) {
	log.Printf("UpdateClientDetails called for %s", req.ClientName)

	// Check if the user exists
	var found bool
	for _, user := range db.Server.Entries {
		encryptedPassword := sha256.Sum256([]byte(req.Password))
		hashedPassword := hex.EncodeToString(encryptedPassword[:])
		if user.Username == req.ClientName && user.Password == hashedPassword {
			found = true
			break
		}
	}
	if !found {
		log.Printf("Client not found, check the username and password")
		return &clgw.UpdateResponse{
			UpdateSuccess: false,
		}, nil
	}

	// get the list of bank servers in etcd
	resp, err := s.etcdClient.Get(ctx, banksPrefix, clientv3.WithPrefix())
	if err != nil {
		log.Printf("Failed to get banks: %v", err)
	}

	// update the client details in all the bank servers
	success := true
	for i := 0; i < len(req.BankNames); i++ {
		bank_name := req.BankNames[i]
		success_ := false
		for _, kv := range resp.Kvs {
			bank := strings.Split(string(kv.Key), "/")[3]
			log.Printf("Bank: %s", bank)
			log.Printf("Bank Name: %s", bank_name)
			log.Printf("Bank Address: %s", string(kv.Value))

			bank_address := string(kv.Value)
			if bank == bank_name && bank_address != "" {
				success_ = true
			}
		}
		if !success_ {
			success = false
			break
		}
	}

	if !success {
		log.Printf("Not all banks are available")
		return &clgw.UpdateResponse{
			UpdateSuccess: false,
			UpdateMessage: "Not all banks are available",
		}, nil
	} else {
		success = true
		for i := 0; i < len(req.BankNames); i++ {
			bank_name := req.BankNames[i]
			for _, kv := range resp.Kvs {
				bank := strings.Split(string(kv.Key), "/")[3]
				bank_address := string(kv.Value)
				if bank == bank_name {
					conn, err := grpc.DialContext(ctx, bank_address, grpc.WithInsecure())
					if err != nil {
						log.Fatalf("Failed to connect to bank: %v", err)
						success = false
						continue
					}
					bankClient := gwbs.NewBankServiceClient(conn)
					var res *gwbs.RegisterResponse
					res, err = bankClient.RegisterClient(ctx, &gwbs.ClientDetails{
						ClientName: req.ClientName,
					})
					if res.RegistrationSuccess == false {
						log.Printf("Failed to register client in bank: %v", err)
						success = false
						break
					}
					if err != nil {
						log.Printf("Failed to update client details in bank: %v", err)
						success = false
					}
					var res2 *gwbs.UpdateResponse
					res2, err = bankClient.UpdateClientDetails(ctx, &gwbs.UpdateDetails{
						ClientName: req.ClientName,
						BankName: bank_name,
						BankBalance: req.BankBalances[i],
					})
					if res2.UpdateSuccess == false {
						log.Printf("Failed to update client details in bank: %v", err)
						success = false
						break
					}
					if err != nil {
						log.Printf("Failed to update client details in bank: %v", err)
						success = false
					}

					conn.Close()
					break
				}
			}
		}
	}

	if success {
		log.Printf("Client details updated successfully")
		return &clgw.UpdateResponse{
			UpdateSuccess: true,
			UpdateMessage: "Client details updated successfully",
		}, nil
	} else {
		log.Printf("All banks are available, but not all banks could be updated")
		return &clgw.UpdateResponse{
			UpdateSuccess: false,
			UpdateMessage: "All banks are available, but not all banks could be updated",
		}, nil
	}
}

// ViewBalance implements the RPC method
func (s *paymentGatewayServer) ViewBalance(ctx context.Context, req *clgw.ViewBalanceRequest) (*clgw.ViewBalanceResponse, error) {
	log.Printf("ViewBalance called with bank: %s, client: %s", req.BankName, req.ClientName)

	// Check if the user exists
	var found bool
	for _, user := range db.Server.Entries {
		encryptedPassword := sha256.Sum256([]byte(req.Password))
		hashedPassword := hex.EncodeToString(encryptedPassword[:])
		if user.Username == req.ClientName && user.Password == hashedPassword {
			found = true
			break
		}
	}
	if !found {
		log.Printf("Client not found, check the username and password")
		return &clgw.ViewBalanceResponse{
			AccountExists: false,
		}, nil
	}

	// get the list of bank servers in etcd
	resp, err := s.etcdClient.Get(ctx, banksPrefix, clientv3.WithPrefix())
	if err != nil {
		log.Printf("Failed to get banks: %v", err)
		return nil, err
	}

	// check if the bank exists
	var bankFound bool
	for _, kv := range resp.Kvs {
		bank := strings.Split(string(kv.Key), "/")[3]
		if bank == req.BankName {
			bankFound = true
			break
		}
	}
	if !bankFound {
		log.Printf("Bank not found")
		return &clgw.ViewBalanceResponse{
			AccountExists: false,
		}, nil
	}

	// if the bank exists, get the address
	var bankAddress string
	for _, kv := range resp.Kvs {
		bank := strings.Split(string(kv.Key), "/")[3]
		if bank == req.BankName {
			bankAddress = string(kv.Value)
			break
		}
	}
	// connect to the bank server
	conn, err := grpc.DialContext(ctx, bankAddress, grpc.WithInsecure())
	if err != nil {
		log.Printf("Failed to connect to bank: %v", err)
		return &clgw.ViewBalanceResponse{
			AccountExists: false,
		}, nil
	}
	defer conn.Close()
	bankClient := gwbs.NewBankServiceClient(conn)
	// call the ViewBalance method on the bank server
	res, err := bankClient.ViewBalance(ctx, &gwbs.ViewBalanceRequest{
		ClientName: req.ClientName,
	})
	if err != nil {
		log.Printf("Failed to view balance: %v", err)
		return &clgw.ViewBalanceResponse{
			AccountExists: false,
		}, nil
	}
	if res.AccountExists {
		log.Printf("Balance: %f", res.AccountBalance)
		return &clgw.ViewBalanceResponse{
			AccountExists: true,
			AccountBalance: fmt.Sprintf("%f", res.AccountBalance),
		}, nil
	} else {
		log.Printf("Account does not exist")
		return &clgw.ViewBalanceResponse{
			AccountExists: false,
		}, nil
	}
}

// InitiateTransaction implements the RPC method
func (s *paymentGatewayServer) InitiateTransaction(ctx context.Context, req *clgw.InitiateTransactionRequest) (*clgw.InitiateTransactionResponse, error) {
	log.Printf("InitiateTransaction called for bank: %s, client: %s", req.BankName, req.ClientName)

	// Check if the user exists
	var found bool
	for _, user := range db.Server.Entries {
		encryptedPassword := sha256.Sum256([]byte(req.Password))
		hashedPassword := hex.EncodeToString(encryptedPassword[:])
		if user.Username == req.ClientName && user.Password == hashedPassword {
			found = true
			break
		}
	}
	if !found {
		log.Printf("Client not found, check the username and password")
		return &clgw.InitiateTransactionResponse{
			TransactionSuccess: false,
			TransactionMessage: "Client not found, check the username and password",
		}, nil
	}

	// check if the transaction id already exists
	for _, transaction := range db.Server.Transactions {
		if transaction.TransactionId == req.TransactionId && transaction.ClientName == req.ClientName {
			log.Printf("Transaction ID already exists")
			return &clgw.InitiateTransactionResponse{
				TransactionSuccess: true,
				TransactionMessage: "Transaction ID already exists",
			}, nil
		}
	}

	// get the list of bank servers in etcd
	resp, err := s.etcdClient.Get(ctx, banksPrefix, clientv3.WithPrefix())
	if err != nil {
		log.Printf("Failed to get banks: %v", err)
		return nil, err
	}

	// check if both banks exist
	var bankFound1, bankFound2 bool
	for _, kv := range resp.Kvs {
		bank := strings.Split(string(kv.Key), "/")[3]
		if bank == req.BankName {
			bankFound1 = true
		}
		if bank == req.BankName2 {
			bankFound2 = true
		}
	}
	if !bankFound1 || !bankFound2 {
		log.Printf("Bank not found")
		return &clgw.InitiateTransactionResponse{
			TransactionSuccess: false,
			TransactionMessage: "Bank not found",
		}, nil
	}

	// if the banks exist, get the addresses
	var bankAddress1, bankAddress2 string
	for _, kv := range resp.Kvs {
		bank := strings.Split(string(kv.Key), "/")[3]
		if bank == req.BankName {
			bankAddress1 = string(kv.Value)
		}
		if bank == req.BankName2 {
			bankAddress2 = string(kv.Value)
		}
	}

	// connect to the bank servers
	conn1, err := grpc.DialContext(ctx, bankAddress1, grpc.WithInsecure())
	if err != nil {
		log.Printf("Failed to connect to bank: %v", err)
		return &clgw.InitiateTransactionResponse{
			TransactionSuccess: false,
			TransactionMessage: "Failed to connect to bank",
		}, nil
	}
	defer conn1.Close()
	bankClient1 := gwbs.NewBankServiceClient(conn1)
	conn2, err := grpc.DialContext(ctx, bankAddress2, grpc.WithInsecure())
	if err != nil {
		log.Printf("Failed to connect to bank: %v", err)
		return &clgw.InitiateTransactionResponse{
			TransactionSuccess: false,
			TransactionMessage: "Failed to connect to bank",
		}, nil
	}
	defer conn2.Close()
	bankClient2 := gwbs.NewBankServiceClient(conn2)

	// check if transaction is possible
	amount, err := strconv.ParseFloat(req.Amount, 32)
	if err != nil {
		log.Printf("Failed to parse amount: %v", err)
		return &clgw.InitiateTransactionResponse{
			TransactionSuccess: false,
			TransactionMessage: "Failed to parse amount",
		}, nil
	}

	res, err := bankClient1.LockTransaction(ctx, &gwbs.TransactionCheckRequest{
		ClientName: req.ClientName,
		BankName: req.BankName,
		TransactionId: req.TransactionId,
		TransactionType: "SEND",
		Amount: float32(amount),
	})
	if err != nil {
		log.Printf("Failed to lock transaction: %v", err)
		return &clgw.InitiateTransactionResponse{
			TransactionSuccess: false,
			TransactionMessage: "Failed to lock transaction",
		}, nil
	}
	if !res.TransactionLock {
		log.Printf("Transaction not possible")
		return &clgw.InitiateTransactionResponse{
			TransactionSuccess: false,
			TransactionMessage: "Transaction not possible",
		}, nil
	}

	res, err = bankClient2.LockTransaction(ctx, &gwbs.TransactionCheckRequest{
		ClientName: req.ClientName2,
		BankName: req.BankName2,
		TransactionId: req.TransactionId,
		TransactionType: "RECEIVE",
		Amount: float32(amount),
	})
	if err != nil {
		// abort the transaction
		_, err1 := bankClient1.AbortTransaction(ctx, &gwbs.AbortTransactionRequest{
			ClientName: req.ClientName,
			BankName: req.BankName,
			TransactionId: req.TransactionId,
		})
		if err1 != nil {
			log.Printf("Failed to abort transaction: %v", err1)
		}

		log.Printf("Failed to lock transaction: %v", err)
		return &clgw.InitiateTransactionResponse{
			TransactionSuccess: false,
			TransactionMessage: "Failed to lock transaction",
		}, nil
	}
	if !res.TransactionLock {
		// abort the transaction
		_, err1 := bankClient1.AbortTransaction(ctx, &gwbs.AbortTransactionRequest{
			ClientName: req.ClientName,
			BankName: req.BankName,
			TransactionId: req.TransactionId,
		})
		if err1 != nil {
			log.Printf("Failed to abort transaction: %v", err1)
		}

		log.Printf("Transaction not possible")
		return &clgw.InitiateTransactionResponse{
			TransactionSuccess: false,
			TransactionMessage: "Transaction not possible",
		}, nil
	}

	// deduct the amount from the client's account
	var res2 *gwbs.InitiateTransactionResponse
	res2, err = bankClient1.InitiateTransaction(ctx, &gwbs.InitiateTransactionRequest{
		ClientName: req.ClientName,
		BankName: req.BankName,
		TransactionType: "SEND",
		Amount: float32(amount),
		TransactionId: req.TransactionId,
	})
	if err != nil {
		log.Printf("Failed to initiate transaction: %v", err)
		return &clgw.InitiateTransactionResponse{
			TransactionSuccess: false,
			TransactionMessage: "Failed to initiate transaction",
		}, nil
	}
	if !res2.TransactionSuccess {
		log.Printf("Failed to initiate transaction")
		return &clgw.InitiateTransactionResponse{
			TransactionSuccess: false,
			TransactionMessage: "Failed to initiate transaction",
		}, nil
	}

	// add the amount to the beneficiary's account
	res2, err = bankClient2.InitiateTransaction(ctx, &gwbs.InitiateTransactionRequest{
		ClientName: req.ClientName2,
		BankName: req.BankName2,
		TransactionType: "RECEIVE",
		Amount: float32(amount),
		TransactionId: req.TransactionId,
	})
	if err != nil {
		log.Printf("Failed to initiate transaction: %v", err)
		return &clgw.InitiateTransactionResponse{
			TransactionSuccess: false,
			TransactionMessage: "Partial transaction",
		}, nil
	}
	if !res2.TransactionSuccess {
		log.Printf("Failed to initiate transaction")
		return &clgw.InitiateTransactionResponse{
			TransactionSuccess: false,
			TransactionMessage: "Partial transaction",
		}, nil
	}

	// update the server database
	db.Server.Transactions = append(db.Server.Transactions, Transaction{
		ClientName: req.ClientName,
		BankName: req.BankName,
		Amount:	float32(amount),
		ClientName2: req.ClientName2,
		BankName2: req.BankName2,
		TransactionId: req.TransactionId,
	})
	data, err := json.Marshal(db)
	if err != nil {
		log.Fatalf("Failed to marshal server database: %v", err)
	}
	if err := ioutil.WriteFile("cmd/gateway/server.json", data, 0644); err != nil {
		log.Fatalf("Failed to write server database: %v", err)
	}

	log.Printf("Transaction successful")
	return &clgw.InitiateTransactionResponse{
		TransactionSuccess: true,
		TransactionMessage: "Transaction successful",
	}, nil
}

// ViewTransactionHistory implements the RPC method
func (s *paymentGatewayServer) ViewTransactionHistory(ctx context.Context, req *clgw.ViewTransactionHistoryRequest) (*clgw.ViewTransactionHistoryResponse, error) {
	log.Printf("ViewTransactionHistory called for %s", req.ClientName)

	// Check if the user exists
	var found bool
	for _, user := range db.Server.Entries {
		encryptedPassword := sha256.Sum256([]byte(req.Password))
		hashedPassword := hex.EncodeToString(encryptedPassword[:])
		if user.Username == req.ClientName && user.Password == hashedPassword {
			found = true
			break
		}
	}
	if !found {
		log.Printf("Client not found, check the username and password")
		return &clgw.ViewTransactionHistoryResponse{
			Transactions: nil,
		}, nil
	}

	// check gateway transactions
	var transactions []*clgw.Transaction
	for _, transaction := range db.Server.Transactions {
		if transaction.ClientName == req.ClientName || transaction.ClientName2 == req.ClientName {
			transactions = append(transactions, &clgw.Transaction{
				ClientName: transaction.ClientName,
				BankName: transaction.BankName,
				Amount: fmt.Sprintf("%f", transaction.Amount),
				ClientName2: transaction.ClientName2,
				BankName2: transaction.BankName2,
				TransactionId: transaction.TransactionId,
			})
		}
	}

	log.Printf("Transactions history for %s", req.ClientName)
	return &clgw.ViewTransactionHistoryResponse{
		Transactions: transactions,
	}, nil
}

// Authentication Interceptor
func AuthInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	// Extract peer information
	p, ok := peer.FromContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "missing peer info")
	}

	// Check if TLS credentials exist
	tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "invalid TLS credentials")
	}

	if len(tlsInfo.State.PeerCertificates) == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "missing client certificate")
	}

	clientCert := tlsInfo.State.PeerCertificates[0]

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		md = metadata.New(map[string]string{})
	} else {
		// Clone the metadata to avoid modifying the original
		md = md.Copy()
	}

	// Add certificate subject to the metadata
	md.Append("cert-subject", clientCert.Subject.CommonName)

	// Create a new context with the updated metadata
	newCtx := metadata.NewIncomingContext(ctx, md)

	// If the authorization is admin, register the user as an admin
	authority := md.Get("cert-subject")

	// Rest of your authentication logic
	username := md.Get("username")
	if len(username) == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "username is not provided")
	}

	password := md.Get("password")
	if len(password) == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "password is not provided")
	}
	encryptedPassword := sha256.Sum256([]byte(password[0]))
	hashedPassword := hex.EncodeToString(encryptedPassword[:])
	log.Printf("Username: %s, Password: %s", username[0], password[0])

	if authority[0] != "StrifeAdmin" {
		// Check if the user exists in the database
		var found bool
		for _, user := range db.Server.Entries {
			if user.Username == username[0] && user.Password == hashedPassword {
				found = true
				break
			}
		}
		if !found {
			return nil, status.Errorf(codes.Unauthenticated, "invalid username or password")
		}
	}

	// split the function to get the service and method
	function := info.FullMethod
	split := strings.Split(function, "/")
	method := split[2]

	if authority[0] != "StrifeAdmin" && (method == "RegisterClient" || method == "UpdateClientDetails") {
		return nil, status.Errorf(codes.PermissionDenied, "admin is not allowed to access this method")
	} else if authority[0] == "StrifeAdmin" && (method == "ViewBalance" || method == "InitiateTransaction" || method == "ViewTransactionHistory") {
		return nil, status.Errorf(codes.PermissionDenied, "client is not allowed to access this method")
	}

	// Continue the execution of the handler with the new context containing certificate info
	return handler(newCtx, req)
}

// Setup mTLS credentials
func setupMTLS() (credentials.TransportCredentials, error) {
	// Load server certificate and key
	serverCert, err := tls.LoadX509KeyPair("cmd/gateway/server.crt", "cmd/gateway/server.key")
	if err != nil {
		return nil, err
	}

	// Load CA certificate for verifying client certificates
	caCert, err := ioutil.ReadFile("ca.crt")
	if err != nil {
		return nil, err
	}

	// Create a certificate pool and add the CA certificate
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caCert) {
		return nil, status.Errorf(codes.Internal, "failed to add CA certificate")
	}

	// Create TLS configuration
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,
	}

	return credentials.NewTLS(tlsConfig), nil
}

// Main function
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

	// Setup mTLS credentials
	creds, err := setupMTLS()
	if err != nil {
		log.Fatalf("Failed to setup TLS: %v", err)
	}

	// Read the server database
	data, err := ioutil.ReadFile("cmd/gateway/server.json")
	if err != nil {
		log.Fatalf("Failed to read server database: %v", err)
	}
	if err := json.Unmarshal(data, &db); err != nil {
		log.Fatalf("Failed to unmarshal server database: %v", err)
	}

	// Create a gRPC server with mTLS and authentication interceptor
	server := grpc.NewServer(
		grpc.Creds(creds),
		grpc.UnaryInterceptor(AuthInterceptor),
	)

	// Listen to the network
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Create a payment gateway server
	paymentGatewayServer := &paymentGatewayServer{
		etcdClient: etcdClient,
	}

	// Register the payment gateway service
	clgw.RegisterPaymentGatewayServiceServer(server, paymentGatewayServer)
	log.Println("Payment gateway service is running on port 50051...")

	// Start the server
	if err := server.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
