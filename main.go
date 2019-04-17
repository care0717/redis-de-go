

package main


import (
    "net"
    "fmt"
    "bufio"
    "strconv"
    "strings"
)

var memory = make(map[string]string, 5000)

func main() {
  ln, err := net.Listen("tcp", ":6379")
  if err != nil {
  	// handle error
  }
  for {
  	conn, err := ln.Accept()
  	if err != nil {
  		// handle error
  	}
  	go handleConnection(conn)
  }
}

func handleConnection(conn net.Conn){
  commands, _ := readConn(conn)
  response := execCommand(commands)
  conn.Write([]byte(response + "\r\n"))
  conn.Close()
}

func readConn(conn net.Conn) ([]string, int){
  r := bufio.NewReader(conn)
  line, err := r.ReadString('\n')
  if err != nil {
    fmt.Println(err)
    conn.Close()
  }
  if line[0] != '*' {
    conn.Write([]byte("-Error missing start char\r\n"))
    conn.Close()
  }
  len, err := strconv.Atoi(strings.TrimRight(line[1:], "\r\n"))
  if err != nil {
    fmt.Println(err)
    conn.Close()
  }
  buf := make([]string, len, len)
  for i := 0; i < len; i++ {
    line, err := r.ReadString('\n')
    if err != nil {
      fmt.Println(err)
      conn.Close()
    }
    if line[0] != '$' {
      conn.Write([]byte("-Error missing start char\r\n"))
      conn.Close()
    }
    line, err = r.ReadString('\n')
    if err != nil {
      fmt.Println(err)
      conn.Close()
    }
    buf[i] = strings.ToLower(strings.TrimRight(line, "\r\n"))
  }
  return buf, len
}

func execCommand(commands []string) (string) {
  command := commands[0]
  switch command {
  case "ping": return "+PONG"
  case "set": return set(commands[1:])
  case "get": return get(commands[1])
  default: return "-Error undefined command "+command
  }
}

func set(keyValue []string) (string) {
  if len(keyValue) != 2 {
    return "-Error invalid key value set"
  }
  memory[keyValue[0]]=keyValue[1]
  return "+OK"
}

func get(key string) (string) {
   fmt.Println(strconv.Itoa(len(v)))
   if ok {
     return "$" + strconv.Itoa(len(v)) + "\r\n" + v
   } else {
     return "-Error unset key"
   }
}
