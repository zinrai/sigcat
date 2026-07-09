# sigcat

A master/worker daemon that reloads a text file on SIGHUP.

## Description

`sigcat` runs a master process and a configurable number of worker processes. The master owns the config file: it prints the contents on startup and, when it receives SIGHUP, reloads the file and displays the updated content. Workers form the process tree and do not read the file. When a worker receives SIGHUP it only logs that it was signaled, so the master and the workers can be signaled independently.

All processes share the `sigcat` name and every worker's parent is the master, so `sigcat` works as a target for tools that locate a master by its process tree.

## Usage

Use default config.txt with 4 workers:

```bash
$ ./sigcat
```

Specify a custom file and worker count:

```bash
$ ./sigcat -file /path/to/myfile.txt -workers 2
```

## Signals

Master:

- `SIGHUP`: Reload the config file and display its contents
- `SIGINT` / `SIGTERM`: Stop the workers with SIGKILL, then shut down

Worker:

- `SIGHUP`: Log that the worker was signaled
- `SIGINT` / `SIGTERM`: Ignored

Workers ignore SIGINT and SIGTERM so that a stray signal cannot reduce the worker count, so the master holds the count without a respawn loop. Each worker is tied to the master's lifetime: if the master dies by any means, including SIGKILL, the kernel sends the worker SIGKILL, so stopping the master always stops its workers and no orphan is left behind.

## Example

1. Create a config file:

```bash
$ echo "Hello, World!" > config.txt
```

2. Start sigcat:

```bash
$ ./sigcat
```

3. In another terminal, update the file:

```bash
$ echo "Updated content!" > config.txt
```

4. Reload by sending SIGHUP to the master:

```bash
$ pkill -HUP -o sigcat
```

`-o` picks the oldest matching process, which is the master. To signal a worker instead, target its PID directly with `kill -HUP <pid>`.

## License

This project is licensed under the [MIT License](./LICENSE).
