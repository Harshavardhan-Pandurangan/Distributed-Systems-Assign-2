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
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()

	for {
		// Ask user what he wants to do
		fmt.Println("What do you want to do?")
		fmt.Println("1. Register Client")
		fmt.Println("2. Update Client")
		fmt.Println("3. View Balance")
		fmt.Println("4. Initiate Transaction")
		fmt.Println("5. View Transaction History")
		fmt.Println("6. Exit")

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
				log.Fatalf("RegisterClient failed: %v", err)
			}
			log.Printf("Registration success: %v", registerResp.RegistrationSuccess)
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
			// bankbalances := make([]string, n)
			bankbalances := make([]float32, n)
			for i := 0; i < n; i++ {
				fmt.Println("Enter bank name:")
				var bankname string
				fmt.Scanln(&bankname)
				fmt.Println("Enter money in bank:")
				var money string
				fmt.Scanln(&money)
				banknames[i] = bankname
				// bankbalances[i] = money
				moneyf, err := strconv.ParseFloat(money, 32)
				if err != nil {
					log.Fatalf("Error converting money to float: %v", err)
				}
				bankbalances[i] = float32(moneyf)
			}

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
				log.Fatalf("UpdateClient failed: %v", err)
			}
			log.Printf("Update success: %v", updateResp.UpdateSuccess)
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
				log.Fatalf("ViewBalance failed: %v", err)
			}
			log.Printf("Balance: %s, Account exists: %v", viewBalanceResp.AccountBalance, viewBalanceResp.AccountExists)
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

			md := metadata.New(map[string]string{
				"username": username,
				"password": password,
			})
			ctx = metadata.NewOutgoingContext(ctx, md)

			// check through the client database and generate a unique transaction id
			var transaction_id string
			for {
				transaction_id = strconv.Itoa(rand.Intn(1000000000))
				// check if transaction id already exists
				transaction_exists := false
				for _, transaction := range s.database.Bank.Transactions {
					if transaction.TransactionId == transaction_id {
						transaction_exists = true
						break
					}
				}
				if !transaction_exists {
					break
				}
			}

			initTransResp, err := client.InitiateTransaction(ctx, &clgw.InitiateTransactionRequest{
				ClientName:		username,
				Password:		password,
				BankName:		bankname,
				Amount:			amount,
				ClientName2:	username2
				BankName2:		bankname2,
				TransactionId:	transaction_id,
			})
			if err != nil {
				log.Fatalf("InitiateTransaction failed: %v", err)
			}

			// if transaction successful, update the database
			if initTransResp.TransactionSuccess {
				// update the database
				transaction := Transaction{
					ClientName:		username,
					BankName:		bankname,
					Amount:			amount,
					ClientName2:	username2,
					BankName2:		bankname2,
					TransactionId:	transaction_id,
				}
				s.database.Bank.Transactions = append
				(s.database.Bank.Transactions, transaction)

				// Save the updated database
				data, err := json.Marshal(s.database)
				if err != nil {
					log.Fatalf("Failed to marshal bank database: %v", err)
				}

				// Save the updated database to file
				err = ioutil.WriteFile("database.json", data, 0644)
				if err != nil {
					log.Fatalf("Failed to write bank database: %v", err)
				}

				log.Printf("Database updated")
			}

			log.Printf("Transaction status: %v, Message: %s", initTransResp.TransactionSuccess, initTransResp.TransactionMessage)

		case 5:
			// Call ViewTransactionHistory RPC
			fmt.Println("Enter username:")
			var username string
			fmt.Scanln(&username)
			fmt.Println("Enter password:")
			var password string
			fmt.Scanln(&password)

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
				log.Fatalf("ViewTransactionHistory failed: %v", err)
			}

			log.Printf("Transaction history:")
			for i, tx := range viewTransResp.Transactions {
				var transaction_type string
				if tx.TransactionType == "SEND" {
					transaction_type = "sent to"
				} else {
					transaction_type = "received from"
				}
				if
				log.Printf("  %d. [ClientName: %s, BankName: %s], Amount %s %s [ClientName: %s, BankName: %s]: TransactionId: %s",
				i+1, tx.ClientName, tx.BankName, tx.Amount, transaction_type, tx.ClientName2, tx.BankName2, tx.TransactionId)
			}
		case 6:
			return
		}
	}
}
