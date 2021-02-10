package main

import (
	"container/list"
	"errors"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"../tw/twrpc"
)

type TaskType int

const (
	ONCE TaskType = iota
	LOOP
	UNKNOWN
)

const (
	MAX_SECOND = 60
	MAX_MINUTE = 60
	MAX_HOUR   = 24
	MAX_DAY    = 30
	MAX_MONTH  = 12
)

type TickTickTask struct {
	taskName string
	taskType TaskType
	schedule []int
}

type TimeWheel struct {
	wheelSize      int
	layerNum       int
	previousPos    int
	overTimeWheel  *TimeWheel
	belowTimeWheel *TimeWheel
	tickTick       *TickTick
	bucket         []*list.List
	lock           sync.Mutex
}

func (t *TimeWheel) putTask(task *TickTickTask, pos int) bool {
	t.lock.Lock()
	defer t.lock.Unlock()
	if pos > t.wheelSize {
		log.Fatal("pos large than the bucket size!")
		return false
	}
	if pos == -1 {
		pos = rand.Intn(t.wheelSize)
	}
	taskList := t.bucket[pos]
	if taskList == nil {
		taskList = list.New()
	}
	taskList.PushBack(&task)
	log.Printf("put task %v %v into wheel %v", task.taskName, task.schedule, t.layerNum)
	return true
}

func (t *TimeWheel) tick() {
	t.lock.Lock()
	defer t.lock.Unlock()
	pos := t.previousPos + 1
	log.Printf("Tick! %v", pos)
	// exceed the max, move to first one, loop the wheel.
	// also send notify to over wheel, which also need a tick.
	if pos >= t.wheelSize {
		if t.overTimeWheel != nil {
			go t.overTimeWheel.tick()
		}
		pos = 0
	}

	taskList := t.bucket[pos]
	if t.belowTimeWheel != nil {
		// if this wheel has below one, means task need to check small scale
		for taskEle := taskList.Front(); taskEle != nil; taskEle = taskEle.Next() {
			task := *taskEle.Value.(**TickTickTask)
			go t.belowTimeWheel.putTask(task, task.schedule[t.belowTimeWheel.layerNum])
		}
	} else {
		for task := taskList.Front(); task != nil; task = task.Next() {
			tickTickTask := task.Value.(**TickTickTask)
			go t.exec(*tickTickTask)
		}
	}
	taskList.Init()
	t.previousPos = pos
}

func (timeWheel *TimeWheel) exec(t *TickTickTask) {
	log.Printf("exec task %v", t)
	timeWheel.tickTick.installTask(t)
}

type TickTick struct {
	clocks      []*TimeWheel
	yearWheel   TimeWheel
	monthWheel  TimeWheel
	dayWheel    TimeWheel
	hourWheel   TimeWheel
	minuteWheel TimeWheel
	secondWheel TimeWheel
}

func (t *TickTick) tick() {
	for true {
		time.Sleep(time.Second)
		time := make([]int, 6)
		for i, timeWheel := range t.clocks {
			time[i] = timeWheel.previousPos
		}
		log.Printf("last time: %v", time)
		t.secondWheel.tick()
	}
}

func main() {
	t := MakeTickTick()
	for t.done() == false {
		time.Sleep(time.Second)
	}
	time.Sleep(time.Second)
}

func (t *TickTick) server() {
	rpc.Register(t)
	rpc.HandleHTTP()

	sockname := twrpc.ServerSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error!", e)
	}
	go http.Serve(l, nil)
}

func (t *TickTick) done() bool {
	return false
}

func MakeTickTick() *TickTick {
	t := TickTick{}
	t.clocks = make([]*TimeWheel, 6)

	t.secondWheel = TimeWheel{
		wheelSize:   60,
		layerNum:    0,
		previousPos: 0,
		bucket:      make([]*list.List, 60),
	}

	t.minuteWheel = TimeWheel{
		wheelSize:      60,
		layerNum:       1,
		previousPos:    0,
		bucket:         make([]*list.List, 60),
		belowTimeWheel: &t.secondWheel,
	}

	t.secondWheel.overTimeWheel = &t.minuteWheel

	t.hourWheel = TimeWheel{
		wheelSize:      24,
		layerNum:       2,
		previousPos:    0,
		bucket:         make([]*list.List, 24),
		belowTimeWheel: &t.minuteWheel,
	}

	t.minuteWheel.overTimeWheel = &t.hourWheel

	t.dayWheel = TimeWheel{
		wheelSize:      30,
		layerNum:       3,
		previousPos:    0,
		bucket:         make([]*list.List, 30),
		belowTimeWheel: &t.hourWheel,
	}

	t.hourWheel.overTimeWheel = &t.dayWheel

	t.monthWheel = TimeWheel{
		wheelSize:      12,
		layerNum:       4,
		previousPos:    0,
		bucket:         make([]*list.List, 12),
		belowTimeWheel: &t.dayWheel,
	}

	t.dayWheel.overTimeWheel = &t.monthWheel

	t.yearWheel = TimeWheel{
		wheelSize:      100,
		layerNum:       5,
		previousPos:    0,
		bucket:         make([]*list.List, 100),
		belowTimeWheel: &t.monthWheel,
	}

	t.monthWheel.overTimeWheel = &t.yearWheel

	t.clocks[0] = &t.secondWheel
	t.clocks[1] = &t.minuteWheel
	t.clocks[2] = &t.hourWheel
	t.clocks[3] = &t.dayWheel
	t.clocks[4] = &t.monthWheel
	t.clocks[5] = &t.yearWheel

	for _, timeWheel := range t.clocks {
		timeWheel.tickTick = &t
		for i := range timeWheel.bucket {
			timeWheel.bucket[i] = list.New()
		}
	}

	t.server()
	go t.tick()
	return &t
}

func (t *TickTick) PutTask(request *twrpc.TimeTaskRequest, response *twrpc.TimeTaskResponse) error {
	log.Printf("get request: name %v, schedule %v", request.Name, request.TimeSchedule)
	jobname := request.Name
	response.Name = jobname
	schedule := request.TimeSchedule
	scheduleArr := strings.Split(schedule, " ")
	if len(scheduleArr) != 6 {
		return errors.New("schedule format should be 6!")
	}
	task := TickTickTask{
		taskName: request.Name,
	}
	task.schedule = make([]int, 6)
	for i := 5; i >= 0; i-- {
		if scheduleArr[i] == "*" {
			task.schedule[i] = -1
			continue
		}
		if time, err := strconv.Atoi(scheduleArr[i]); err == nil {
			task.schedule[i] = time
		} else {
			return errors.New("schedule should be * or number")
		}
	}
	t.installTask(&task)
	return nil
}

func (t *TickTick) installTask(task *TickTickTask) {
	findFirstPos := false
	for i := 5; i >= 0; i-- {
		if task.schedule[i] == -1 {
			if !findFirstPos {
				continue
			} else {
				task.schedule[i] = rand.Intn(t.clocks[i].wheelSize)
				continue
			}
		} else {
			if findFirstPos {
				continue
			} else {
				tasks := t.clocks[i].bucket[task.schedule[i]]
				if tasks == nil {
					tasks = list.New()
					t.clocks[i].bucket[task.schedule[i]] = tasks
				}
				t.clocks[i].putTask(task, task.schedule[i])
				findFirstPos = true
			}
		}
	}
	if !findFirstPos {
		task.schedule[0] = rand.Intn(t.clocks[0].wheelSize)
		t.clocks[0].putTask(task, task.schedule[0])
	}
	log.Printf("install task: %v", task)
}
