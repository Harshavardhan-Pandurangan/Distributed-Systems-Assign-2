#   Distributed MapReduce Implementation

This project implements a distributed MapReduce system with a master and worker nodes. It supports Word Count and Inverted Index tasks.

##   Prerequisites

* gRPC-go

##   Installation

1.  Install dependencies:

```bash
go mod tidy
```

##   Usage

1.  Start the master:

```bash
go run cmd/master/main.go -reducers=<num_reducers> -task=<task_type> -input=<input_folder>
```

* `<num_reducers>`: Number of reduce tasks.
* `<task_type>`: Task to perform, either "wordcount" or "invertedindex".
* `<input_folder>`: Folder containing input files.

Example:

```bash
go run cmd/master/main.go -reducers=3 -task=wordcount -input=input
```

2.  Start the workers (in separate terminals):

```bash
go run cmd/worker/main.go -master=<master_address> -type=<worker_type>
```

* `<master_address>`: Address of the master (e.g., localhost:50051).
* `<worker_type>`: Type of the worker, either "mapper" or "reducer".

Examples:

```bash
go run cmd/worker/main.go -master=localhost:50051 -type=mapper
go run cmd/worker/main.go -master=localhost:50051 -type=reducer
```

##   Configuration

* **Number of reduce tasks:**　This is set via the `-reducers` command-line argument when starting the master.
* **Task type:**　This is set via the `-task` command-line argument when starting the master ("wordcount" or "invertedindex").
* **Input folder:**　This is set via the `-input` command-line argument when starting the master. The input folder should contain the text files to be processed.
* **Master address:**　This is provided to the workers via the `-master` command-line argument.
* **Worker type:**　This is provided to the workers via the `-type` command-line argument ("mapper" or "reducer").

##   Tasks

* **Word Count:**
        * Input: A set of text files in the input folder.
        * Output: Files in the `datasets/output` folder, each containing a list of words and their counts.
* **Inverted Index:**
        * Input: A set of text files in the input folder.
        * Output: Files in the `datasets/output` folder, each containing a list of words and the filenames in which they appear.

##   I/O Format

* **Intermediate files:** Mappers create intermediate files in the `datasets/intermediate` directory. The files are named `intermediate-<worker_id>-<reduce_task_id>`. Each line in the intermediate file contains a key-value pair separated by a space. For Word Count, the format is "word count", and for Inverted Index, it's "word filename".
* **Output files:** Reducers create output files in the `datasets/output` directory. The files are named with the reducer's worker ID. Each line contains a key-value pair. For Word Count, the format is "word total\_count", and for Inverted Index, it's "word list\_of\_filenames" (filenames separated by spaces).