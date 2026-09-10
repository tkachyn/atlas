package protocol

import (
	"fmt"
	"strconv"
	"strings"
)

// command represents one parsed atlas request
type Command struct {
	Name string
	Args []string
}

// parse converts one line of client input into a command
func Parse(line string) (Command, error) {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return Command{}, fmt.Errorf("empty command")
	}

	name := strings.ToUpper(fields[0])
	args := fields[1:]

	switch name {
	case "PING":
		if len(args) != 0 {
			return Command{}, fmt.Errorf("PING does not accept arguments")
		}
	case "SET":
		if len(args) != 2 {
			return Command{}, fmt.Errorf("SET requires key and value")
		}
	case "GET", "DEL", "EXISTS", "TTL":
		if len(args) != 1 {
			return Command{}, fmt.Errorf("%s requires key", name)
		}
	case "EXPIRE":
		if len(args) != 2 {
			return Command{}, fmt.Errorf("EXPIRE requires key and seconds")
		}
		if _, err := strconv.ParseInt(args[1], 10, 64); err != nil {
			return Command{}, fmt.Errorf("EXPIRE seconds must be an integer")
		}
	case "EXPIREAT":
		if len(args) != 2 {
			return Command{}, fmt.Errorf("EXPIREAT requires key and timestamp")
		}
		if _, err := strconv.ParseInt(args[1], 10, 64); err != nil {
			return Command{}, fmt.Errorf("EXPIREAT timestamp must be an integer")
		}
	default:
		return Command{}, fmt.Errorf("unknown command %q", fields[0])
	}

	return Command{Name: name, Args: args}, nil
}

// error formats an error response for the atlas protocol
func Error(message string) string {
	return "ERR " + message + "\n"
}

// format serializes a parsed command for persistence
func Format(cmd Command) string {
	return strings.Join(append([]string{cmd.Name}, cmd.Args...), " ")
}

// mutates reports whether a command changes stored state
func Mutates(cmd Command) bool {
	switch cmd.Name {
	case "SET", "DEL", "EXPIRE", "EXPIREAT":
		return true
	default:
		return false
	}
}
