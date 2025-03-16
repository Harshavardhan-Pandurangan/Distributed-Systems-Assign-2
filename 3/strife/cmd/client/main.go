package main

import (
	"context"
	"flag"
	"log"
	"io/ioutil"
	"fmt"
	"crypto/tls"
	"crypto/x509"
	"time"
	"strconv"
	"math/rand"
	"encoding/json"

	clgw "strife/proto/cl-gw"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

type Transaction struct {
	ClientName string `json:"client_name"`
	BankName string `json:"bank_name"`
	Amount float32 `json:"amount"`
	ClientName2 string `json:"client_name2"`
	BankName2 string `json:"bank_name2"`
	TransactionId string `json:"transaction_id"`
}

type ClientDatabase struct {
	Client struct {
		Name string `json:"name"`
		Transactions []Transaction `json:"transactions"`
	} `json:"client"`
}

var clientID string
var clientDatabase ClientDatabase
var online bool = true
var requestQueue []string


// Load mTLS credentials
func loadCredentials(clientID string) (credentials.TransportCredentials, error) {
	// Load the CA certificate
	caCert, err := ioutil.ReadFile("ca.crt")
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %v", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to add CA's certificate")
	}

	// Load the client certificate and key
	clientCert, err := tls.LoadX509KeyPair(
		fmt.Sprintf("cmd/client/certs/%s.crt", clientID),
		fmt.Sprintf("cmd/client/certs/%s.key", clientID),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %v", err)
	}

	// Create the TLS credentials
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      caCertPool,
	}

	return credentials.NewTLS(tlsConfig), nil
}

// Case 1: Register Client
func registerClient(ctx context.Context, client clgw.PaymentGatewayServiceClient, username string, password string) {
	md := metadata.New(map[string]string{
		"username": username,
		"password": password,
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	registerResp, err := client.RegisterClient(ctx, &clgw.ClientDetails{
		ClientName: username,
		Password:   password,
	})
	if err != nil {
		log.Printf("RegisterClient failed: %v", err)
	}
	log.Printf("Registration success: %v", registerResp.RegistrationSuccess)
}

// Case 2: Update Client
func updateClient(ctx context.Context, client clgw.PaymentGatewayServiceClient, username string, password string, banknames []string, bankbalances []float32) {
	md := metadata.New(map[string]string{
		"username": username,
		"password": password,
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	updateResp, err := client.UpdateClientDetails(ctx, &clgw.UpdateDetails{
		ClientName:	  username,
		Password:     password,
		BankNames:    banknames,
		BankBalances: bankbalances,
	})
	if err != nil {
		log.Printf("UpdateClient failed: %v", err)
	}
	log.Printf("Update success: %v", updateResp.UpdateSuccess)
}

// Case 3: View Balance
func viewBalance(ctx context.Context, client clgw.PaymentGatewayServiceClient, username string, password string, bankname string) {
	md := metadata.New(map[string]string{
		"username": username,
		"password": password,
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	viewBalanceResp, err := client.ViewBalance(ctx, &clgw.ViewBalanceRequest{
		ClientName:	username,
		Password:	password,
		BankName:	bankname,
	})
	if err != nil {
		log.Printf("ViewBalance failed: %v", err)
	}
	log.Printf("Balance: %s, Account exists: %v", viewBalanceResp.AccountBalance, viewBalanceResp.AccountExists)
}

// Case 4: Initiate Transaction
func initiateTransaction(ctx context.Context, client clgw.PaymentGatewayServiceClient, username string, password string, bankname string, username2 string, bankname2 string, amount string, transaction_id string) {
	md := metadata.New(map[string]string{
		"username": username,
		"password": password,
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	initTransResp, err := client.InitiateTransaction(ctx, &clgw.InitiateTransactionRequest{
		ClientName:		username,
		Password:		password,
		BankName:		bankname,
		Amount:			amount,
		ClientName2:	username2,
		BankName2:		bankname2,
		TransactionId:	transaction_id,
	})
	if err != nil {
		log.Printf("InitiateTransaction failed: %v", err)
	}

	// if transaction successful, update the database
	amountf, err := strconv.ParseFloat(amount, 32)
	if err != nil {
		log.Fatalf("Error converting amount to float: %v", err)
	}
	if initTransResp.TransactionSuccess {
		// update the database
		transaction := Transaction{
			ClientName:		username,
			BankName:		bankname,
			Amount:			float32(amountf),
			ClientName2:	username2,
			BankName2:		bankname2,
			TransactionId:	transaction_id,
		}
		clientDatabase.Client.Transactions = append(clientDatabase.Client.Transactions, transaction)

		// Save the updated database
		data, err := json.Marshal(clientDatabase)
		if err != nil {
			log.Fatalf("Failed to marshal client database: %v", err)
		}

		// Save the updated database to file
		err = ioutil.WriteFile(fmt.Sprintf("cmd/client/databases/%s.json", clientID), data, 0644)
		if err != nil {
			log.Fatalf("Failed to write client database: %v", err)
		}

		log.Printf("Database updated")
	} else if initTransResp.TransactionMessage == "Partial transaction" {
		// queue the transaction
		log.Printf("Partial transaction. Request is in queue")
		request := fmt.Sprintf("4 %s %s %s %s %s %s %s", username, password, bankname, username2, bankname2, amount, transaction_id)
		requestQueue = append(requestQueue, request)
	}

	log.Printf("Transaction status: %v, Message: %s", initTransResp.TransactionSuccess, initTransResp.TransactionMessage)
}

// Case 5: View Transaction History
func viewTransactionHistory(ctx context.Context, client clgw.PaymentGatewayServiceClient, username string, password string) {
	md := metadata.New(map[string]string{
		"username": username,
		"password": password,
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	viewTransResp, err := client.ViewTransactionHistory(ctx, &clgw.ViewTransactionHistoryRequest{
		ClientName:	username,
		Password:	password,
	})
	if err != nil {
		log.Printf("ViewTransactionHistory failed: %v", err)
	}

	log.Printf("Transaction history:")
	for i, tx := range viewTransResp.Transactions {
		var transaction_type string
		if tx.ClientName == username {
			transaction_type = "sent to"
		} else {
			transaction_type = "received from"
			var temp string
			temp = tx.BankName
			tx.BankName = tx.BankName2
			tx.BankName2 = temp
			temp = tx.ClientName
			tx.ClientName = tx.ClientName2
			tx.ClientName2 = temp
		}
		log.Printf("  %d. [ClientName: %s, BankName: %s], Amount %s %s [ClientName: %s, BankName: %s]: TransactionId: %s",
		i+1, tx.ClientName, tx.BankName, tx.Amount, transaction_type, tx.ClientName2, tx.BankName2, tx.TransactionId)
	}

	log.Printf("Transaction history end")
}

// Function to handle requests in the queue
func handleRequests(ctx context.Context, client clgw.PaymentGatewayServiceClient, requestQueue *[]string) {
	for {
		if !online {
			// Sleep for a while
			time.Sleep(2 * time.Second)
			continue
		}

		// Process the request queue
		for len(*requestQueue) > 0 {
			// Parse the request
			var choice int
			fmt.Sscanf((*requestQueue)[0], "%d", &choice)

			switch choice {
			case 1:
				// Call RegisterClient RPC
				var username string
				var password string
				fmt.Sscanf((*requestQueue)[0], "%d %s %s", &choice, &username, &password)
				registerClient(ctx, client, username, password)
			case 2:
				// Call UpdateClient RPC
				var username string
				var password string
				var banknames []string
				var bankbalances []float32
				fmt.Sscanf((*requestQueue)[0], "%d %s %s %v %v", &choice, &username, &password, &banknames, &bankbalances)
				updateClient(ctx, client, username, password, banknames, bankbalances)
			case 3:
				// Call ViewBalance RPC
				var username string
				var password string
				var bankname string
				fmt.Sscanf((*requestQueue)[0], "%d %s %s %s", &choice, &username, &password, &bankname)
				viewBalance(ctx, client, username, password, bankname)
			case 4:
				// Call InitiateTransaction RPC
				var username string
				var password string
				var bankname string
				var username2 string
				var bankname2 string
				var amount string
				var transaction_id string
				fmt.Sscanf((*requestQueue)[0], "%d %s %s %s %s %s %s %s", &choice, &username, &password, &bankname, &username2, &bankname2, &amount, &transaction_id)
				initiateTransaction(ctx, client, username, password, bankname, username2, bankname2, amount, transaction_id)
			case 5:
				// Call ViewTransactionHistory RPC
				var username string
				var password string
				fmt.Sscanf((*requestQueue)[0], "%d %s %s", &choice, &username, &password)
				viewTransactionHistory(ctx, client, username, password)
			}

			// Remove the request from the queue
			*requestQueue = (*requestQueue)[1:]
		}
	}
}

// Main function
func main() {
	// Parse command line flags
	clientID := flag.String("id", "client1", "Client ID")
	flag.Parse()

	// Print the client info
	log.Printf("Client ID: %s", *clientID)

	// Get the mTLS credentials
	creds, err := loadCredentials(*clientID)
	if err != nil {
		log.Fatalf("Failed to load credentials: %v", err)
	}

	// Connect to the server
	log.Printf("Connecting to payment gateway at localhost:50051")
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Create a client
	client := clgw.NewPaymentGatewayServiceClient(conn)

	// Create a context with metadata for authentication
	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
	defer cancel()

	// Load the database
	data, err := ioutil.ReadFile(fmt.Sprintf("cmd/client/databases/%s.json", *clientID))
	if err != nil {
		log.Fatalf("Failed to read client database: %v", err)
	}

	// Unmarshal the JSON data
	clientDatabase = ClientDatabase{}
	err = json.Unmarshal(data, &clientDatabase)
	if err != nil {
		log.Fatalf("Failed to unmarshal client database: %v", err)
	}

	// System status
	log.Printf("System online")

	// start a go routine to handle requests in the queue
	go handleRequests(ctx, client, &requestQueue)

	// Main loop
	for {
		// Ask user what he wants to do
		fmt.Println("What do you want to do?")
		fmt.Println("1. Register Client")
		fmt.Println("2. Update Client")
		fmt.Println("3. View Balance")
		fmt.Println("4. Initiate Transaction")
		fmt.Println("5. View Transaction History")
		fmt.Println("6. System offline")
		fmt.Println("7. System online")
		fmt.Println("8. Exit")

		var choice int
		fmt.Scanln(&choice)

		switch choice {
		case 1:
			// Call RegisterClient RPC
			fmt.Println("Enter username:")
			var username string
			fmt.Scanln(&username)
			fmt.Println("Enter password:")
			var password string
			fmt.Scanln(&password)

			if online {
				registerClient(ctx, client, username, password)
			} else {
				log.Printf("System offline. Request is in queue")
				request := fmt.Sprintf("1 %s %s", username, password)
				requestQueue = append(requestQueue, request)
			}
		case 2:
			// Call UpdateClient RPC
			fmt.Println("Enter username:")
			var username string
			fmt.Scanln(&username)
			fmt.Println("Enter password:")
			var password string
			fmt.Scanln(&password)
			// ask for list of bank ids followed by money in each bank
			fmt.Println("Enter number of banks:")
			var n int
			fmt.Scanln(&n)
			banknames := make([]string, n)
			bankbalances := make([]float32, n)
			for i := 0; i < n; i++ {
				fmt.Println("Enter bank name:")
				var bankname string
				fmt.Scanln(&bankname)
				fmt.Println("Enter money in bank:")
				var money string
				fmt.Scanln(&money)
				banknames[i] = bankname
				moneyf, err := strconv.ParseFloat(money, 32)
				if err != nil {
					log.Fatalf("Error converting money to float: %v", err)
				}
				bankbalances[i] = float32(moneyf)
			}

			if online {
				updateClient(ctx, client, username, password, banknames, bankbalances)
			} else {
				log.Printf("System offline. Request is in queue")
				request := fmt.Sprintf("2 %s %s %v %v", username, password, banknames, bankbalances)
				requestQueue = append(requestQueue, request)
			}
		case 3:
			// Call ViewBalance RPC
			fmt.Println("Enter username:")
			var username string
			fmt.Scanln(&username)
			fmt.Println("Enter password:")
			var password string
			fmt.Scanln(&password)
			var bankname string
			fmt.Println("Enter bank name:")
			fmt.Scanln(&bankname)

			if online {
				viewBalance(ctx, client, username, password, bankname)
			} else {
				log.Printf("System offline. Request is in queue")
				request := fmt.Sprintf("3 %s %s %s", username, password, bankname)
				requestQueue = append(requestQueue, request)
			}
		case 4:
			// Call InitiateTransaction RPC
			fmt.Println("Enter username:")
			var username string
			fmt.Scanln(&username)
			fmt.Println("Enter password:")
			var password string
			fmt.Scanln(&password)
			fmt.Println("Enter bank name:")
			var bankname string
			fmt.Scanln(&bankname)
			fmt.Println("Enter beneficiary username:")
			var username2 string
			fmt.Scanln(&username2)
			fmt.Println("Enter beneficiary bank name:")
			var bankname2 string
			fmt.Scanln(&bankname2)
			fmt.Println("Enter amount:")
			var amount string
			fmt.Scanln(&amount)

			// check through the client database and generate a unique transaction id
			var transaction_id string
			for {
				transaction_id = strconv.Itoa(rand.Intn(1000000000))
				// check if transaction id already exists
				transaction_exists := false
				for _, transaction := range clientDatabase.Client.Transactions {
					if transaction.TransactionId == transaction_id {
						transaction_exists = true
						break
					}
				}
				if !transaction_exists {
					break
				}
			}

			if online {
				initiateTransaction(ctx, client, username, password, bankname, username2, bankname2, amount, transaction_id)
			} else {
				log.Printf("System offline. Request is in queue")
				request := fmt.Sprintf("4 %s %s %s %s %s %s %s", username, password, bankname, username2, bankname2, amount, transaction_id)
				requestQueue = append(requestQueue, request)
			}
		case 5:
			// Call ViewTransactionHistory RPC
			fmt.Println("Enter username:")
			var username string
			fmt.Scanln(&username)
			fmt.Println("Enter password:")
			var password string
			fmt.Scanln(&password)

			if online {
				viewTransactionHistory(ctx, client, username, password)
			} else {
				log.Printf("System offline. Request is in queue")
				request := fmt.Sprintf("5 %s %s", username, password)
				requestQueue = append(requestQueue, request)
			}
		case 6:
			// System go offline
			log.Printf("System going offline")
			online = false
		case 7:
			// System go online
			log.Printf("System going online")
			online = true
		case 8:
			return
		}
	}
}
