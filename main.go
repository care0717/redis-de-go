package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	stdtime "time"

	"github.com/care0717/redis-de-go/syncmap"

	"github.com/care0717/redis-de-go/resp"
)

var memory = syncmap.SyncMap{}

func main() {
	var (
		host string
		port int
	)
	flag.StringVar(&host, "host", "localhost", "Listen ip address")
	flag.IntVar(&port, "port", 6379, "Listen port")
	flag.Parse()
	addr := fmt.Sprintf("%s:%d", host, port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Printf("listening on %s\n", addr)
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println(err)
		}
		fmt.Printf("connection succeeded from %s\n", conn.RemoteAddr())
		go func(conn net.Conn) {
			err := handleConnection(conn)
			if err != nil {
				fmt.Println(err)
			}
		}(conn)
	}
}

func handleConnection(conn net.Conn) error {
	defer conn.Close()
	for {
		commands, err := readConn(conn)
		if err == io.EOF {
			break
		}
		writer := bufio.NewWriter(conn)
		if err != nil {
			_, err = writer.Write([]byte(resp.Error(err.Error()).String()))
			if err != nil {
				return err
			}
			err = writer.Flush()
			if err != nil {
				return err
			}
			continue
		}
		response := execCommand(commands)
		_, err = writer.Write([]byte(response.String()))
		if err != nil {
			return err
		}
		err = writer.Flush()
		if err != nil {
			return err
		}
	}
	return nil
}

func readConn(conn net.Conn) ([]string, error) {
	r := bufio.NewReader(conn)
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if line[0] != '*' {
		return nil, errors.New("missing start char `*`")
	}
	length, err := strconv.Atoi(strings.TrimRight(line[1:], "\r\n"))
	if err != nil {
		return nil, errors.New("missing array length")
	}
	buf := make([]string, length)
	for i := 0; i < length; i++ {
		line, err := r.ReadString('\n')

		if err != nil {
			return nil, err
		}
		if line[0] != '$' {
			return nil, errors.New("missing start char `$`")
		}

		line, err = r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		buf[i] = strings.ToLower(strings.TrimRight(line, "\r\n"))
	}
	return buf, nil
}

func execCommand(commands []string) resp.RESP {
	command := commands[0]
	switch command {
	case "ping":
		return ping(commands[1:])
	case "set":
		return set(commands[1:])
	case "get":
		return get(commands[1])
	case "del":
		return del(commands[1:])
	case "exists":
		return exists(commands[1])
	case "incrby", "decrby":
		key, diff, err := indecFormat(commands)
		if err != nil {
			return err
		}
		return changeValue(key, diff)
	case "incr", "decr":
		if len(commands) != 2 {
			return resp.Error("wrong number of arguments")
		}
		key := commands[1]
		var diff int
		if command == "incr" {
			diff = 1
		} else {
			diff = -1
		}
		return changeValue(key, diff)
	case "rename":
		return rename(commands[1:])
	case "time":
		return time()
	case "append":
		return append(commands[1:])
	case "dbsize":
		return dbsize()
	case "touch":
		return touch(commands[1:])
	case "mget":
		return multiGet(commands[1:])
	case "mset":
		return multiSet(commands[1:])
	default:
		return resp.Error("undefined command " + command)
	}
}

func multiGet(keys []string) resp.RESP {
	if len(keys) == 0 {
		return resp.Error("wrong number of arguments for 'mget' command")
	}
	values := make(resp.Array, len(keys))
	for i, key := range keys {
		values[i] = get(key)
	}
	return values
}

func multiSet(keyValues []string) resp.RESP {
	if len(keyValues) == 0 || len(keyValues)%2 == 1 {
		return resp.Error("wrong number of arguments for 'mset' command")
	}
	for i := 0; i < len(keyValues); i += 2 {
		key := keyValues[i]
		value := keyValues[i+1]
		set([]string{key, value})
	}
	return resp.SimpleString("OK")
}

func touch(keys []string) resp.RESP {
	if len(keys) == 0 {
		return resp.Error("wrong number of arguments for 'touch' command")
	}
	var counter int

	for _, key := range keys {
		if _, ok := memory.Load(key); ok {
			counter += 1
		}
	}

	return resp.Integer(counter)
}

func dbsize() resp.RESP {
	return resp.Integer(len(memory.Keys()))
}

func append(keyValue []string) resp.RESP {
	if len(keyValue) != 2 {
		return resp.Error("wrong number of arguments for 'append' command")
	}

	key := keyValue[0]
	value := keyValue[1]
	v, _ := memory.Load(key)
	newValue := v + value
	memory.Store(key, newValue)
	return resp.Integer(len(newValue))
}

func time() resp.RESP {
	now := stdtime.Now().UnixNano() / 1000
	timestamp := resp.BulkString(strconv.FormatInt(now/1000000, 10))
	micro := resp.BulkString(strconv.FormatInt(now%1000000, 10))
	return resp.Array{timestamp, micro}
}

func rename(keyNames []string) resp.RESP {
	if len(keyNames) != 2 {
		return resp.Error("wrong number of arguments for 'rename' command")
	}

	if memory.Rename(keyNames[0], keyNames[1]) {
		return resp.SimpleString("OK")
	} else {
		return resp.Error("no such key")
	}

}

func ping(echo []string) resp.RESP {
	if len(echo) > 0 {
		return resp.SimpleString(echo[0])
	} else {
		return resp.SimpleString("PONG")
	}
}

func set(keyValue []string) resp.RESP {
	if len(keyValue) == 2 {
		memory.Store(keyValue[0], keyValue[1])
		return resp.SimpleString("OK")
	} else if len(keyValue) == 3 {
		option := keyValue[2]
		_, ok := memory.Load(keyValue[0])
		switch option {
		case "nx":
			{
				if ok {
					return resp.BulkString("")
				} else {
					memory.Store(keyValue[0], keyValue[1])
					return resp.SimpleString("OK")
				}
			}
		case "xx":
			{
				if !ok {
					return resp.BulkString("")
				} else {
					memory.Store(keyValue[0], keyValue[1])
					return resp.SimpleString("OK")
				}
			}
		default:
			return resp.Error("invalid option")
		}
	} else {
		return resp.Error("invalid key value set")
	}
}

func get(key string) resp.RESP {
	v, ok := memory.Load(key)
	if ok {
		return resp.BulkString(v)
	} else {
		return resp.BulkString("")
	}
}

func del(keys []string) resp.RESP {
	count := 0
	for _, key := range keys {
		if _, ok := memory.Load(key); ok {
			memory.Delete(key)
			count += 1
		}
	}
	return resp.Integer(count)
}

func exists(key string) resp.RESP {
	count := 0
	if _, ok := memory.Load(key); ok {
		count = 1
	}
	return resp.Integer(count)
}

func indecFormat(commands []string) (string, int, resp.RESPError) {
	if len(commands) != 3 {
		return "", 0, resp.Error("wrong number of arguments")
	}

	key := commands[1]
	diff := commands[2]

	num, err := strconv.Atoi(diff)
	if err != nil {
		return "", 0, resp.Error("value is not an integer or out of range")
	}
	if commands[0] == "decrby" {
		num = -num
	}
	return key, num, nil
}

func changeValue(key string, diff int) resp.RESP {
	v, ok := memory.Load(key)
	if !ok {
		memory.Store(key, strconv.Itoa(diff))
		return resp.Integer(diff)
	}

	val, err := strconv.Atoi(v)
	if err != nil {
		return resp.Error("value is not an integer or out of range")
	}

	result := val + diff
	memory.Store(key, strconv.Itoa(result))
	return resp.Integer(result)
}
