import os
import subprocess
from time import sleep

if __name__ == '__main__':
    # Ensure required directories exist
    os.makedirs('datasets/intermediate', exist_ok=True)
    os.makedirs('datasets/output', exist_ok=True)
    os.makedirs('datasets/input', exist_ok=True)

    # Clear existing files
    for folder in ['datasets/intermediate', 'datasets/output', 'datasets/input']:
        for file in os.listdir(folder):
            os.remove(os.path.join(folder, file))

    # Take all required inputs
    num_reducers = int(input('Enter the number of reducers: '))
    task = input('Enter the task to be performed (wordcount, invertedindex): ')
    input_folder = input('Enter the input folder name: ')

    # Copy input files
    for file in os.listdir(input_folder):
        os.system(f'cp {input_folder}/{file} datasets/input/')

    # Run the master node (allowing its output to be displayed)
    master_cmd = f'go run cmd/master/main.go -task {task} -reducers {num_reducers} -input {input_folder}'
    master_process = subprocess.Popen(master_cmd, shell=True)

    # sleep for 2 seconds to allow master to start
    sleep(2)

    # Run worker nodes in parallel (suppressing output)
    for file in os.listdir(input_folder):
        worker_cmd = 'go run cmd/worker/main.go'
        subprocess.Popen(worker_cmd, shell=True,
                         stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)

    # Run reducer nodes in parallel (suppressing output)
    for i in range(num_reducers):
        reducer_cmd = 'go run cmd/worker/main.go -type reducer'
        subprocess.Popen(reducer_cmd, shell=True,
                         stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)

    print("All worker and reducer nodes started.")
