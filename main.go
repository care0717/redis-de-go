package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	stdtime "time"
)

type RESPBulkString string

func (b RESPBulkString) String() string {
	if l := len(b); l > 0 {
		return "$" + strconv.Itoa(len(b)) + "\r\n" + string(b) + "\r\n"
	} else {
		return "$-1\r\n"
	}
}

type RESPSimpleString string

func (b RESPSimpleString) String() string {
	return "+" + string(b) + "\r\n"
}

type RESPError string

func (e RESPError) String() string {
	return "-ERROR " + string(e) + "\r\n"
}

type RESPInteger int

func (i RESPInteger) String() string {
	return ":" + strconv.Itoa(int(i)) + "\r\n"
}

type Response interface {
	String() string
}

type RESPArray []Response

func (a RESPArray) String() string {
	l := len(a)
	res := "*" + strconv.Itoa(l) + "\r\n"
	for i := 0; i < l; i++ {
		res += a[i].String()
	}
	return res
}

type INDEC int

const (
	INC INDEC = iota
	DEC
)

var memory = SyncMap{}

func main() {
	port := "6379"
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("Listning on port " + port)
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println(err)
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	for {
		commands, err := readConn(conn)
		if err == io.EOF {
			break
		}
		if err != nil {
			conn.Write([]byte(err.Error()))
		}
		response := execCommand(commands)
		conn.Write([]byte(response.String()))
	}
	//fmt.Println("conn close")
	conn.Close()
}

func readConn(conn net.Conn) ([]string, error) {
	r := bufio.NewReader(conn)
	line, err := r.ReadString('\n')
	if err != nil {
		return make([]string, 1, 1), err
	}
	if line[0] != '*' {
		return make([]string, 1, 1), errors.New(RESPError("missing start char").String())
	}
	len, err := strconv.Atoi(strings.TrimRight(line[1:], "\r\n"))
	if err != nil {
		return make([]string, 1, 1), errors.New(RESPError("missing array number").String())
	}
	buf := make([]string, len, len)
	for i := 0; i < len; i++ {
		line, err := r.ReadString('\n')

		if err != nil {
			return make([]string, 1, 1), err
		}
		if line[0] != '$' {
			return make([]string, 1, 1), errors.New(RESPError("missing start char").String())
		}

		line, err = r.ReadString('\n')
		if err != nil {
			return make([]string, 1, 1), err
		}
		buf[i] = strings.ToLower(strings.TrimRight(line, "\r\n"))
	}
	return buf, nil
}

func execCommand(commands []string) Response {
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
	case "incrby":
		return changeValue(commands[1:], INC)
	case "decrby":
		return changeValue(commands[1:], DEC)
	case "rename":
		return rename(commands[1:])
	case "time":
		return time()
	default:
		return RESPError("undefined command " + command)
	}
}

func time() Response {
	now := stdtime.Now().UnixNano() / 1000
	timestamp := RESPBulkString(strconv.FormatInt(now/1000000, 10))
	micro := RESPBulkString(strconv.FormatInt(now%1000000, 10))
	return RESPArray{timestamp, micro}
}

func rename(keyNames []string) Response {
	if len(keyNames) == 2 {
		if memory.Rename(keyNames[0], keyNames[1]) {
			return RESPSimpleString("OK")
		} else {
			return RESPError("no such key")
		}
	} else {
		return RESPError("wrong number of arguments for 'rename' command")
	}
}

func ping(echo []string) Response {
	if len(echo) > 0 {
		return RESPSimpleString(echo[0])
	} else {
		return RESPSimpleString("PONG")
	}
}

func set(keyValue []string) Response {
	if len(keyValue) == 2 {
		memory.Store(keyValue[0], keyValue[1])
		return RESPSimpleString("OK")
	} else if len(keyValue) == 3 {
		option := keyValue[2]
		_, ok := memory.Load(keyValue[0])
		switch option {
		case "nx":
			{
				if ok {
					return RESPBulkString("")
				} else {
					memory.Store(keyValue[0], keyValue[1])
					return RESPSimpleString("OK")
				}
			}
		case "xx":
			{
				if !ok {
					return RESPBulkString("")
				} else {
					memory.Store(keyValue[0], keyValue[1])
					return RESPSimpleString("OK")
				}
			}
		default:
			return RESPError("invalid option")
		}
	} else {
		return RESPError("invalid key value set")
	}
}

func get(key string) Response {
	v, ok := memory.Load(key)
	if ok {
		return RESPBulkString(v)
	} else {
		return RESPError("unset key")
	}
}

func del(keys []string) Response {
	count := 0
	for _, key := range keys {
		if _, ok := memory.Load(key); ok {
			memory.Delete(key)
			count += 1
		}
	}
	return RESPInteger(count)
}

func exists(key string) Response {
	count := 0
	if _, ok := memory.Load(key); ok {
		count = 1
	}
	return RESPInteger(count)
}

func changeValue(keyDiffs []string, indec INDEC) Response {
	if len(keyDiffs) < 2 {
		return RESPError("wrong number of arguments")
	}

	key := keyDiffs[0]
	diff := keyDiffs[1]

	num, err := strconv.Atoi(diff)
	if err != nil {
		return RESPError("value is not an integer or out of range")
	}

	v, ok := memory.Load(key)
	if !ok {
		memory.Store(key, diff)
		return RESPInteger(num)
	}

	val, err := strconv.Atoi(v)
	if err != nil {
		return RESPError("value is not an integer or out of range")
	}

	var result int
	switch indec {
	case INC:
		{
			result = val + num
		}
	case DEC:
		{
			result = val - num
		}
	}
	memory.Store(key, strconv.Itoa(result))
	return RESPInteger(result)
}
