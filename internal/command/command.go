package command

import (
	"strconv"
	"time"

	"github.com/tkachyn/atlas/internal/protocol"
	"github.com/tkachyn/atlas/internal/store"
)

// execute applies a parsed command to the store and formats its response
func Execute(cmd protocol.Command, data *store.Store) string {
	switch cmd.Name {
	case "SET":
		data.Set(cmd.Args[0], cmd.Args[1])
		return "OK\n"
	case "GET":
		value, ok := data.Get(cmd.Args[0])
		if !ok {
			return "(nil)\n"
		}
		return value + "\n"
	case "DEL":
		if data.Delete(cmd.Args[0]) {
			return "1\n"
		}
		return "0\n"
	case "EXISTS":
		if data.Exists(cmd.Args[0]) {
			return "1\n"
		}
		return "0\n"
	case "EXPIRE":
		seconds, err := strconv.ParseInt(cmd.Args[1], 10, 64)
		if err != nil {
			return protocol.Error("EXPIRE seconds must be an integer")
		}
		if data.Expire(cmd.Args[0], seconds) {
			return "1\n"
		}
		return "0\n"
	case "EXPIREAT":
		nanoseconds, err := strconv.ParseInt(cmd.Args[1], 10, 64)
		if err != nil {
			return protocol.Error("EXPIREAT timestamp must be an integer")
		}
		if data.ExpireAt(cmd.Args[0], time.Unix(0, nanoseconds)) {
			return "1\n"
		}
		return "0\n"
	case "TTL":
		return strconv.FormatInt(data.TTL(cmd.Args[0]), 10) + "\n"
	default:
		return protocol.Error("unknown command " + cmd.Name)
	}
}
