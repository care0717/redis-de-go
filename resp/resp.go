package resp

import (
	"strconv"
)

type BulkString string

func (b BulkString) String() string {
	if l := len(b); l > 0 {
		return "$" + strconv.Itoa(len(b)) + "\r\n" + string(b) + "\r\n"
	} else {
		return "$-1\r\n"
	}
}

type SimpleString string

func (b SimpleString) String() string {
	return "+" + string(b) + "\r\n"
}

type RESPError interface {
	String() string
	Error() string
}
type Error string

func (e Error) String() string {
	return "-ERROR " + string(e) + "\r\n"
}
func (e Error) Error() string {
	return string(e)
}

type Integer int

func (i Integer) String() string {
	return ":" + strconv.Itoa(int(i)) + "\r\n"
}

type RESP interface {
	String() string
}

type Array []RESP

func (a Array) String() string {
	l := len(a)
	res := "*" + strconv.Itoa(l) + "\r\n"
	for i := 0; i < l; i++ {
		res += a[i].String()
	}
	return res
}
