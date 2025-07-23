// - fsq or "fixed size queue" is a queue of tasks. Tasks are functions that are to be run 
// concurrently. The max size of the queue, and max number of concurrent tasks are determined
// at the time which the fsq is created.
// 
// - Internally, fsq uses a ring buffer for O(1) adding and removing to the queue.
// 
// - Additionally, tasks are re-used (when possible) to remove the overhead of creating new tasks in memory
// for each item added to the queue.
// 
// - Use Queue.Add() to add tasks to the queue. The queue will automatically perform the tasks added to it
// as tasks are added and as tasks are completed. In other words, just add to the queue using .Add(), and 
// the queue will do the rest.
// 
// - To prevent duplication, new tasks cannot be added with the same id as tasks that are waiting in the queue.
// However, there is no logic to prevent duplicating a task that has already been removed from the queue (processed).
// 
// IMPORTANT: Adding to the queue is a fire and forget operation. There is no feedback regarding if a 
// task has been completed successfully or not.


package fsq

import "fmt"
import "errors"
import "strings"

type FixedSizeQueue struct {
	Name string
	items *ringBuffer
	tasksById map[int]*task
	waitingTasksByExternalId map[string]*task
	readyTaskPool *[]*task
	countProcessing int
	maxProcessing int
	taskCount int
	isRunning bool
}

var Queue *FixedSizeQueue


// @size: the max size of the queue. Defaults to 1 if size of <= 0 is passed in
func Init(size int, name string, maxProcessCount int) *FixedSizeQueue {
	if size <= 0 {
		size = 1
	}

	rbItems := make([]*task, size)
	rb := &ringBuffer{
		MaxSize: size,
		items:   &rbItems,
	}

	queue := FixedSizeQueue{
		Name: name,
		items: rb,
		tasksById: map[int]*task{},
		waitingTasksByExternalId: map[string]*task{},
		readyTaskPool: &[]*task{},
		maxProcessing: maxProcessCount,
	}

	Queue = &queue
	return &queue
}


func(q *FixedSizeQueue) Start() {
	q.isRunning = true
}


func(q *FixedSizeQueue) Stop() {
	q.isRunning = false
}


func(q *FixedSizeQueue) IsRunning() bool {
	return q.isRunning
}


func(q *FixedSizeQueue) Add(action func(params map[string]interface{}) error, params map[string]interface{}, id string) error {
	if !q.isRunning {
		errMsg := fmt.Sprintf("FixedSizeQueue %s is not running. Try starting and then adding.", q.Name)
		return errors.New(errMsg)
	}

	if q.items.IsFull {
		errMsg := fmt.Sprintf("FixedSizeQueue %s has no capacity at this time. Try later.", q.Name)
		return errors.New(errMsg)
	}

	_, err := q.isValidId(id)
	if err != nil {
		return err
	}
	
	var taskToUse *task

	if len(*q.readyTaskPool) > 0 {
		// reuse an existing task from the readyTaskPool and remove the task from the readyTaskPool
		taskToUse = q.popTask(q.readyTaskPool)
	} else {
		//since no existing tasks can be re-used, create a new task
		taskToUse = &task{}
		q.taskCount++
		taskToUse.SetId(q.taskCount)
		q.tasksById[taskToUse.id] = taskToUse
	}

	taskToUse.SetStateWaiting()
	taskToUse.SetAction(action)
	taskToUse.SetParams(params)
	taskToUse.SetExternalId(id)
	q.waitingTasksByExternalId[id] = taskToUse
	q.items.Enqueue(taskToUse)
	q.processTask()
	return nil
}


func (q *FixedSizeQueue) isValidId(id string) (bool, error) {
	// check if empty string
	trimmed := strings.TrimSpace(id)
	if len(trimmed) == 0 {
		return false, errors.New("Id for task is not valid, only uses space characters.")
	}

	// check if tasks with id already exists in waiting tasks
	_, ok := q.waitingTasksByExternalId[id]

	if ok {
		return false, errors.New("Id for task is already waiting to be processed.")
	}

	return true, nil
}


func (q *FixedSizeQueue) processTask() {
	if q.countProcessing >= q.maxProcessing {
		return
	}

	task := q.items.Dequeue()

	if task == nil {
		return
	}

	delete(q.waitingTasksByExternalId, task.externalId)
	q.countProcessing++
	task.SetStateProcessing()
	go q.actionWrapper(task)
}


func (q *FixedSizeQueue) actionWrapper(task *task) {
	err := task.CallAction()

	if err != nil {
		// TODO: log error
	}

	// sets state back to ready state and removes info from task
	task.Clean()
	*q.readyTaskPool = append(*q.readyTaskPool, task)
	q.countProcessing--

	// when the task's action is done, attempt to process the next waiting task
	q.processTask()
}


func (q *FixedSizeQueue) popTask(slice *[]*task) *task {
	tasks := *slice

	length := len(tasks)
	
	if length == 0 {
		return nil
	}

	task := tasks[length - 1]
	*slice = tasks[:length - 1]

	return task
}
