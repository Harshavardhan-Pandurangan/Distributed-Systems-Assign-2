import os
import subprocess
from time import time, sleep
import matplotlib.pyplot as plt

if __name__ == "__main__":
    # Take all the required inputs
    num_servers = int(input("Enter the number of servers: "))
    num_clients = int(input("Enter the number of clients: "))
    s_type = input("Enter the scheduling type (first, rr, ll): ")

    # Start etcd in the background
    etcd_cmd = 'etcd'
    etcd_process = subprocess.Popen(etcd_cmd, shell=True,
                                    stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)

    # Sleep for 2 seconds to allow etcd to start
    sleep(2)

    # Start the master node (Load Balancer) and redirect output to a file
    master_log = "master_log.txt"
    master_cmd = f'go run cmd/lb/main.go -type {s_type} > {master_log} 2>&1'
    master_process = subprocess.Popen(master_cmd, shell=True)

    # Sleep for 2 seconds to allow the master node to start
    sleep(2)

    # Start the server nodes
    for i in range(num_servers):
        server_cmd = f'go run cmd/worker/main.go -id {i} -port {50052 + i}'
        subprocess.Popen(server_cmd, shell=True,
                         stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)

    # Sleep for 2 seconds to allow servers to start
    sleep(2)

    print("All servers started.")

    # Start measuring the total execution time
    start_time = time()

    # Start the client nodes and store their process objects
    client_processes = []
    response_times = []

    for i in range(num_clients):
        client_cmd = f'go run cmd/client/main.go -task="sum-up 1000000000"'
        proc = subprocess.Popen(
            client_cmd, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
        client_processes.append(proc)

    print("All clients started. Waiting for them to finish...")

    # Wait for all clients to finish and collect response times
    for proc in client_processes:
        stdout, stderr = proc.communicate()  # Wait for client to finish
        try:
            last_line = stdout.strip().split("\n")[-1] if stdout else ""
            if "Time taken:" in last_line:
                response_time = float(last_line.split(":")[1].strip().split()[
                                      0][:-1])  # Extract number
                response_times.append(response_time)
        except Exception as e:
            print(f"Error processing client output: {e}")

    # Calculate total execution time
    end_time = time()
    total_time = end_time - start_time

    # Calculate system throughput
    throughput = num_clients / (total_time if total_time > 0 else 1)

    # Display results
    print("\n========= Performance Metrics =========")
    print(f"Total Requests: {num_clients}")
    print(f"Total Execution Time: {total_time:.2f} sec")
    print(f"System Throughput: {throughput:.2f} requests/sec")

    if response_times:
        avg_response_time = sum(response_times) / len(response_times)
        print(f"Average Response Time: {avg_response_time:.2f} ms")
        print(f"Min Response Time: {min(response_times):.2f} ms")
        print(f"Max Response Time: {max(response_times):.2f} ms")
    print("=======================================")

    # Collect load data from the master log file
    load_series = [[] for _ in range(num_servers)]

    print("\nCollecting Load Balancer logs...")

    # Read the log file instead of capturing from Popen
    with open(master_log, "r") as log_file:
        logs = log_file.readlines()

    for log in logs:
        print(f"Master Output: {log.strip()}")  # Debug output
        if "Worker" in log and "load" in log:
            try:
                parts = log.split()
                server_id = int(parts[1])  # Assuming worker ID is second word
                load = float(parts[-1])  # Assuming load is the last value
                load_series[server_id].append(load)
            except ValueError as e:
                print(f"Error parsing log line: {log.strip()} -> {e}")

    # Calculate the mean load on each server
    mean_loads = [sum(loads) / len(loads)
                  if loads else 0 for loads in load_series]
    print("Mean load on each server:", mean_loads)

    # Plot the load on each server over time
    plt.figure()
    for i in range(num_servers):
        plt.plot(load_series[i], label=f'Server {i}')
    plt.xlabel('Time')
    plt.ylabel('Load')
    plt.legend()
    plt.savefig('load.png')
    plt.show()

    # Plot the mean load on each server
    plt.figure()
    plt.bar(range(num_servers), mean_loads)
    plt.xlabel('Server')
    plt.ylabel('Mean Load')
    plt.savefig('mean_load.png')
    plt.show()

    print("Test completed.")

    # Kill the master node
    master_process.kill()

    # Kill the etcd process
    etcd_process.kill()

    # Kill all the server nodes
    for i in range(num_servers):
        os.system(f'fuser -k {50052 + i}/tcp')
