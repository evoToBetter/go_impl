package twrpc

import (
	"os"
	"strconv"
)

type ResponseStatus int

const (
	OK ResponseStatus = iota
	FAIL
	UNKNOWN
)

type TimeTaskRequest struct {
	Name         string
	TimeSchedule string
}

type TimeTaskResponse struct {
	Name   string
	Status ResponseStatus
}

func ServerSock() string {
	s := "/var/tmp/824-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}
