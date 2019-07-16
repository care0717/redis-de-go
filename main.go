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
)

type INDEC int

const (
	INC INDEC = iota
	DEC
)

var memory = New()

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
			conn.Write([]byte(err.Error() + "\r\n"))
		}
		response := execCommand(commands)
		conn.Write([]byte(response + "\r\n"))
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
		return make([]string, 1, 1), errors.New("-Error missing start char")
	}
	len, err := strconv.Atoi(strings.TrimRight(line[1:], "\r\n"))
	if err != nil {
		return make([]string, 1, 1), errors.New("-Error missing array number")
	}
	buf := make([]string, len, len)
	for i := 0; i < len; i++ {
		line, err := r.ReadString('\n')

		if err != nil {
			return make([]string, 1, 1), err
		}
		if line[0] != '$' {
			return make([]string, 1, 1), errors.New("-Error missing start char")
		}

		line, err = r.ReadString('\n')
		if err != nil {
			return make([]string, 1, 1), err
		}
		buf[i] = strings.ToLower(strings.TrimRight(line, "\r\n"))
	}
	return buf, nil
}

func execCommand(commands []string) string {
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
	default:
		return "-Error undefined command " + command
	}
}

func ping(echo []string) string {
	if len(echo) > 0 {
		return "+" + echo[0]
	} else {
		return "+PONG"
	}
}

func set(keyValue []string) string {
	if len(keyValue) == 2 {
		memory.Store(keyValue[0], keyValue[1])
		return "+OK"
	} else if len(keyValue) == 3 {
		option := keyValue[2]
		_, ok := memory.Load(keyValue[0])
		switch option {
		case "nx":
			{
				if ok {
					return "$-1"
				} else {
					memory.Store(keyValue[0], keyValue[1])
					return "+OK"
				}
			}
		case "xx":
			{
				if !ok {
					return "$-1"
				} else {
					memory.Store(keyValue[0], keyValue[1])
					return "+OK"
				}
			}
		default:
			return "-Error invalid option"
		}
	} else {
		return "-Error invalid key value set"
	}
}

func get(key string) string {
	v, ok := memory.Load(key)
	if ok {
		return "$" + strconv.Itoa(len(v)) + "\r\n" + v
	} else {
		return "-Error unset key"
	}
}

func del(keys []string) string {
	count := 0
	for _, key := range keys {
		if _, ok := memory.Load(key); ok {
			memory.Delete(key)
			count += 1
		}
	}
	return ":" + strconv.Itoa(count)
}

func exists(key string) string {
	count := 0
	if _, ok := memory.Load(key); ok {
		count = 1
	}
	return ":" + strconv.Itoa(count)
}

func changeValue(keyDiffs []string, indec INDEC) string {
	if len(keyDiffs) < 2 {
		return "-Error wrong number of arguments"
	}
	key := keyDiffs[0]
	diff := keyDiffs[1]
	v, ok := memory.Load(key)
	if !ok {
		memory.Store(key, diff)
		return ":" + diff
	}
	val, err := strconv.Atoi(v)
	if err != nil {
		return "-Error value is not an integer or out of range"
	}
	num, err := strconv.Atoi(diff)
	if err != nil {
		return "-Error value is not an integer or out of range"
	}
	var result string
	switch indec {
	case INC:
		{
			result = strconv.Itoa(val + num)
		}
	case DEC:
		{
			result = strconv.Itoa(val - num)
		}
	}
	memory.Store(key, result)
	return ":" + result
}
