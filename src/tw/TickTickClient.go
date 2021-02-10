package main

import (
	"log"
	"net/rpc"
	"os"

	"../tw/twrpc"
)

func main() {
	if len(os.Args) < 2 {
		log.Printf("args %v", os.Args)
		log.Fatal("Usage: TickTickClient jobname timeschedule")
		os.Exit(1)
	}
	request := twrpc.TimeTaskRequest{}
	request.Name = os.Args[1]
	request.TimeSchedule = os.Args[2]
	response := twrpc.TimeTaskResponse{}
	callFinish := call("TickTick.PutTask", &request, &response)
	if callFinish {
		log.Printf("put task %v", response.Name)
	} else {
		log.Fatal("could not put task!")
	}
}

func call(rpcname string, args interface{}, reply interface{}) bool {
	sockname := twrpc.ServerSock()
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("could not open sock to server", err)
		return false
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err != nil {
		log.Fatal("could not call TickTickServer", err)
		return false
	}
	return true
}
