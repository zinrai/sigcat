# sigcat

A lightweight daemon that outputs file contents on SIGHUP signal.

## Description

`sigcat` is a daemon process that monitors a text file and outputs its contents to stdout. When receiving a SIGHUP signal, it reloads the file and displays the updated content.

## Installation

```bash
$ go install github.com/zinrai/sigcat@latest
```

## Usage

Use default config.txt:

```bash
$ ./sigcat
```

Specify a custom file:

```bash
$ ./sigcat /path/to/myfile.txt
```

## Signals

- `SIGHUP`: Reload the configuration file and display contents
- `SIGINT` / `SIGTERM`: Gracefully shutdown the daemon

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

4. Send SIGHUP:

```bash
$ pkill -HUP sigcat
```

## License

This project is licensed under the MIT License - see the [LICENSE](https://opensource.org/license/mit) for details.
