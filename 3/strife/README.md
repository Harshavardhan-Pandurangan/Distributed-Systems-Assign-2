#   Simplified Stripe Implementation

This project implements a simplified version of Stripe, a payment gateway. It includes bank servers, clients, and a payment gateway.

##   Prerequisites

* gRPC-go
* etcd
* openssl

##   Installation

1.  Install dependencies:

```bash
go mod tidy
```

##   Usage

1.  Start etcd:

```bash
etcd
```

2.  Start the bank servers (in separate terminals, specifying unique ports and ids):

```bash
go run cmd/bank/main.go -port=<port_number> -id=<bank_id>
```

* `<port_number>`: Port number for the bank server (e.g., 50052, 50053).
* `<bank_id>`: Unique ID for the bank server (e.g., bank0, bank1).

Example:

```bash
go run cmd/bank/main.go -port=50052 -id=bank0
go run cmd/bank/main.go -port=50053 -id=bank1
```

3.  Start the payment gateway:

```bash
go run cmd/gateway/main.go
```

4.  Start the clients (in separate terminals, specifying unique ids):

```bash
go run cmd/client/main.go -id=<client_id>
```

* `<client_id>`: Unique ID for the client (e.g., client1, client2).

Example:

```bash
go run cmd/client/main.go -id=client1
go run cmd/client/main.go -id=client2

go run cmd/client/main.go -id=client0 # for testing purposes, serves as an admin client
```

##   Configuration

* **Bank Servers:**
    * Each bank server needs a unique port and ID.
    * Bank server details are stored in `cmd/bank/databases/<bank_id>.json` files. You can initialize these files with bank-specific data (e.g., user accounts, balances).
* **Payment Gateway:**
    * The payment gateway uses `cmd/gateway/server.json` to store user credentials and transaction history.
    * The payment gateway connects to etcd to discover available bank servers. Ensure etcd is running.
    * mTLS certificates are located in `cmd/gateway/`.
* **Clients:**
    * Each client needs a unique ID.
    * Client data (e.g., transactions) is stored in `cmd/client/databases/<client_id>.json`.
    * mTLS certificates are located in `cmd/client/certs/`.
* **Certificates:**
    * Ensure that the necessary certificates (`ca.crt`, `server.crt`, `server.key`, `client1.crt`, `client1.key`, etc.) are generated and placed in the appropriate directories (`cmd/gateway/`, `cmd/client/certs/`). You can use a tool like `openssl` to generate these.

##   Features

* **Secure Authentication:**
    * Implemented using SSL/TLS mutual authentication.
    * Clients authenticate with the payment gateway using client certificates.
    * The payment gateway verifies client certificates using a Certificate Authority (CA).
* **Authorization:**
    * Implemented using gRPC interceptors.
    * Clients must provide valid credentials (username and password) via metadata.
    * The gateway verifies the credentials against stored data.
    * Role-based authorization is used to restrict access to sensitive operations (e.g., only admins can register clients).
* **Logging:**
    * Implemented using a gRPC interceptor in the payment gateway.
    * Logs include transaction amount, client identification, method name, and errors.
* **Idempotent Payments:**
    * Implemented by checking for existing transaction IDs.
    * The payment gateway checks if a transaction ID has already been processed to prevent multiple deductions.
* **Offline Payments:**
    * Clients queue payments locally when offline.
    * Payments are automatically retried when the client comes back online.
* **2PC with Timeout:**
    * Implemented using the payment gateway as the coordinator and bank servers as voters.
    * The payment gateway locks transactions in involved bank servers.
    * If any bank server fails to acknowledge or if a timeout occurs, the transaction is aborted.

##   gRPC Services

* **PaymentGatewayService (cl-gw.proto):**
    * `RegisterClient`: Registers a new client.
    * `UpdateClientDetails`: Updates client details (e.g., bank accounts, balances).
    * `ViewBalance`: Views the balance of a client's account.
    * `InitiateTransaction`: Initiates a payment transaction.
    * `ViewTransactionHistory`: Views the transaction history of a client.
* **BankService (gw-bank.proto):**
    * `RegisterClient`: Registers a client with the bank.
    * `UpdateClientDetails`: Updates client details at the bank.
    * `ViewBalance`: Views the balance of a client's account at the bank.
    * `LockTransaction`: Checks and locks a transaction.
    * `InitiateTransaction`: Initiates a transaction at the bank.
    * `AbortTransaction`: Aborts a transaction.
