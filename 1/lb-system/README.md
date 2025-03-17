# Load Balancing System

This project implements a client-server system with a look-aside load balancer. It supports Pick First, Round Robin, and Least Load balancing policies.

## Prerequisites

* gRPC-go
* etcd client v3
* gopsutil

## Installation

1.  Install dependencies:

```bash
go mod tidy
```

## Usage

###   1. Start etcd:

Ensure etcd is running. If you have it installed locally, you can usually start it with:

```bash
etcd
```

###   2. Start the LB server:

```bash
go run cmd/lb/main.go -type=<scheduling_type>
```

* `<scheduling_type>` can be `first` (Pick First), `rr` (Round Robin), or `ll` (Least Load). If not provided, the default is `rr`.

Example:

```bash
go run lb_server.go -type=ll
```

###   3. Start the backend servers (in separate terminals):

```bash
go run cmd/worker/main.go -id=<worker_id> -port=<port>
```

    * `<worker_id>` is a unique identifier for the worker (e.g., worker1).
    * `<port>` is the port number for the worker to listen on (e.g., 50052).

    Examples:

```bash
go run cmd/worker/main.go -id=worker1 -port=50052
go run cmd/worker/main.go -id=worker2 -port=50053
go run cmd/worker/main.go -id=worker3 -port=50054
```

###   4. Start the clients (in separate terminals):

```bash
go run client.go -id=<client_id> -lb=<lb_address> -task=<task_data>
```

* `<client_id>` is a unique identifier for the client (e.g., client1).
* `<lb_address>` is the address of the load balancer (e.g., localhost:50051).
* `<task_data>` is the task to send to the worker.
    * If the task starts with "sleep", followed by a number, the worker will sleep for that many seconds (e.g., "sleep 5").
    * If the task starts with "sum-up", followed by a number, the worker will sum up all numbers from 1 to that number (e.g., "sum-up 100").
    * Any other task will be considered unknown.

Examples:

```bash
go run cmd/client/main.go -id=client1 -lb=localhost:50051 -task="sleep 3"
go run cmd/client/main.go -id=client2 -lb=localhost:50051 -task="sum-up 1000000"
go run cmd/client/main.go -id=client3 -lb=localhost:50051 -task="sum-up 100000000"
```

##   Configuration

* **Load balancing policy:**　This is set via the `-type` command-line argument when starting the LB server.
* **Server addresses:**　Workers register themselves with the etcd service, and the LB server retrieves the worker addresses from etcd. The LB server address is provided to the client via the `-lb` command-line argument.

##   Testing

* To simulate high load, run multiple client instances concurrently, sending various tasks.
* Observe the LB server's output to see how it distributes requests to the workers.
* Monitor worker utilization and response times to evaluate the effectiveness of each load balancing policy.
* Use the `setup.py` file to automate the process of starting the servers and clients.