package fsq

import "testing"
import "errors"
import "fmt"
import "time"
import "github.com/stretchr/testify/assert"


// ---------------------------------------------------------------------------
// ---------------------------------------------------------------------------
// TESTING FIXED SIZE QUEUE (fsq.go)
// ---------------------------------------------------------------------------
// ---------------------------------------------------------------------------
func TestInit_WithValidInput(t *testing.T) {
	assert := assert.New(t)

	size := 10
	name := "test-queue"
	maxProcessing := 5

	q := Init(size, name, maxProcessing)

	assert.NotNil(q, "Init should not return nil")

	assert.Equal(name, q.Name, "Queue name should be set correctly")
	assert.Equal(size, q.items.MaxSize, "Queue max size should be set correctly")
	assert.Equal(5, q.maxProcessing, "maxProcessing should be parsed and set as int")

	assert.Empty(q.tasksById, "tasksById should be initialized as empty")
	assert.Empty(q.waitingTasksByExternalId, "waitingTasksByExternalId should be initialized as empty")
	assert.Empty(q.readyTaskPool, "readyTaskPool should be initialized as empty")

	assert.Equal(0, q.countProcessing, "countProcessing should be zero after init")
	assert.Equal(0, q.taskCount, "taskCount should be zero after init")
	assert.False(q.isRunning, "isRunning should be false after init")
}


func TestIsRunning_ReflectsState(t *testing.T) {
	assert := assert.New(t)
	q := Init(10, "test-queue", 5)

	// Initial state
	assert.False(q.IsRunning(), "Expected IsRunning to be false after Init")

	// After Start
	q.Start()
	assert.True(q.IsRunning(), "Expected IsRunning to be true after Start")

	// After Stop
	q.Stop()
	assert.False(q.IsRunning(), "Expected IsRunning to be false after Stop again")
}


func TestIsValidId_EmptyString(t *testing.T) {
	assert := assert.New(t)
	q := Init(10, "test-queue", 5)

	valid, err := q.isValidId("   ")

	assert.False(valid)
	assert.Error(err)
	assert.EqualError(err, "Id for task is not valid, only uses space characters.")
}


func TestIsValidId_IdDoesNotExistInMap(t *testing.T) {
	assert := assert.New(t)
	q := Init(10, "test-queue", 5)

	valid, err := q.isValidId("some-id")

	assert.True(valid)
	assert.NoError(err)
}


func TestIsValidId_IdExistsInMap(t *testing.T) {
	assert := assert.New(t)
	q := Init(10, "test-queue", 5)

	// Manually insert a task into the map
	q.waitingTasksByExternalId["task-123"] = &task{}

	valid, err := q.isValidId("task-123")

	assert.False(valid)
	assert.Error(err)
	assert.EqualError(err, "Id for task is already waiting to be processed.")
}


func TestPopTask_EmptySlice(t *testing.T) {
	assert := assert.New(t)
	q := Init(10, "test-queue", 5)

	tasks := []*task{}

	result := q.popTask(&tasks)

	assert.Nil(result, "Expected nil when popping from empty slice")
	assert.Len(tasks, 0, "Slice should remain empty")
}


func TestPopTask_SingleItem(t *testing.T) {
	assert := assert.New(t)
	q := Init(10, "test-queue", 5)

	t1 := &task{id: 1}
	tasks := []*task{t1}

	result := q.popTask(&tasks)

	assert.Equal(t1, result, "Expected to pop the only task")
	assert.Len(tasks, 0, "Slice should be empty after popping")
}


func TestPopTask_MultipleItems(t *testing.T) {
	assert := assert.New(t)
	q := Init(10, "test-queue", 5)

	t1 := &task{id: 1}
	t2 := &task{id: 2}
	tasks := []*task{t1, t2}

	result := q.popTask(&tasks)

	assert.Equal(t2, result, "Expected to pop the last task")
	assert.Len(tasks, 1, "Slice should have one item left")
	assert.Equal(t1, tasks[0], "Remaining item should be the first task")
}


func TestAdd_ReturnsErrorIfQueueIsRunning(t *testing.T) {
	assert := assert.New(t)
	q := Init(10, "TestQueue", 5)

	err := q.Add(nil, nil, "id-1")

	assert.Error(err)
	assert.Contains(err.Error(), "is not running")
}


func sleeper(params map[string]interface{}) error {
	amt, _ := params["amt"].(int)
	dur := time.Duration(amt)
	time.Sleep(dur * time.Second)
	return nil
}

func TestAdd_ReturnsErrorIfQueueIsFull(t *testing.T) {
	assert := assert.New(t)
	queueName := "TestQueue"
	q := Init(1, queueName, 1)
	q.Start()

	action := sleeper
	params := map[string]interface{}{"amt": 1}

	err1 := q.Add(action, params, "id-1") //1st task is immediately dequed and processing
	err2 := q.Add(action, params, "id-2") //2nd task is stuck waiting in the queue sice only one process allowed
	err3 := q.Add(action, params, "id-3") //3rd task should hit a full queue and be turned away

	assert.NoError(err1)
	assert.NoError(err2)
	assert.Error(err3)
	errMsg := fmt.Sprintf("FixedSizeQueue %s has no capacity at this time. Try later.", queueName)
	assert.EqualError(err3, errMsg)
}


func TestAdd_ReturnsErrorIfIdIsInvalid(t *testing.T) {
	assert := assert.New(t)
	q := Init(10, "TestQueue", 1)
	q.Start()

	err := q.Add(nil, nil, "  ")

	assert.Error(err)
	assert.EqualError(err, "Id for task is not valid, only uses space characters.")

	action := sleeper
	params := map[string]interface{}{"amt": 1}

	//1st task is immediately dequed and processing
	err1 := q.Add(action, params, "id-1")

	//2nd task is stuck waiting in the queue since only one process allowed
	err2 := q.Add(action, params, "id-2")
	
	//3rd task should be allowed in queue since 1st task is no longer in a "waiting" state, even though it has the same id
	err3 := q.Add(action, params, "id-1")
	
	//4th task should be turned away because 2nd task is still in "waiting" state and 4th task has same id as 2nd task
	err4 := q.Add(action, params, "id-2")

	assert.NoError(err1)
	assert.NoError(err2)
	assert.NoError(err3)
	assert.Error(err4)
	assert.EqualError(err4, "Id for task is already waiting to be processed.")
}


func TestAdd_ReusesTaskFromReadyPool(t *testing.T) {
	assert := assert.New(t)
	q := Init(3, "TestQueue", 1)
	q.Start()

	// used to verify that the task's action function actually is called when adding a task to the queue
	wasCalled := false
	callCheckerParams := map[string]interface{}{"key": "val"}
	callChecker := func(params map[string]interface{}) error {
		wasCalled = true
		return nil
	}

	// used to tell how long the processing of the task should take (implemented via time.Sleep)
	sleeperParams := map[string]interface{}{"amt": 1}

	// 1st task is immediately dequeued and processing is immediate.
	// task is placed in the readyTaskPool from a go routine, so need a short sleep before asserting
	err1 := q.Add(callChecker, callCheckerParams, "ext-1")
	assert.NoError(err1)
	assert.Len(q.tasksById, 1)

	time.Sleep(100 * time.Millisecond)
	assert.Len(*q.readyTaskPool, 1)
	assert.True(wasCalled)
	
	// 2nd task is re-used from readyTaskPool and is immediately dequeued and processing takes 1 second
	err2 := q.Add(sleeper, sleeperParams, "ext-2")
	assert.NoError(err2)
	assert.Len(*q.readyTaskPool, 0)
	assert.Len(q.tasksById, 1)
	
	// 3rd task is waiting in queue until 2nd is finished processing
	// Since no tasks are in the readyTaskPool, a new task is created
	err3 := q.Add(sleeper, sleeperParams, "ext-3")
	assert.NoError(err3)
	assert.Len(q.tasksById, 2)
	assert.Len(q.waitingTasksByExternalId, 1)
	assert.Contains(q.waitingTasksByExternalId, "ext-3")
}


// ---------------------------------------------------------------------------
// ---------------------------------------------------------------------------
// TESTING RING BUFFER (ringBuffer.go)
// ---------------------------------------------------------------------------
// ---------------------------------------------------------------------------
func createRingBuffer(size int) *ringBuffer {
	items := make([]*task, size)

	return &ringBuffer{
		MaxSize: size,
		items: &items,
	}
}


func TestEnqueueDequeue_Normal(t *testing.T) {
	assert := assert.New(t)
	rb := createRingBuffer(3)

	t1 := &task{id: 1}
	t2 := &task{id: 2}

	err1 := rb.Enqueue(t1)
	err2 := rb.Enqueue(t2)

	assert.NoError(err1)
	assert.NoError(err2)
	assert.Equal(2, rb.CurrentSize)
	assert.False(rb.IsFull)

	t3 := &task{id: 3}
	err3 := rb.Enqueue(t3)
	assert.NoError(err3)
	assert.Equal(3, rb.CurrentSize)
	assert.True(rb.IsFull)

	out1 := rb.Dequeue()
	assert.Equal(2, rb.CurrentSize)
	assert.False(rb.IsFull)

	out2 := rb.Dequeue()
	out3 := rb.Dequeue()

	assert.Equal(t1, out1)
	assert.Equal(t2, out2)
	assert.Equal(t3, out3)
	assert.Equal(0, rb.CurrentSize)
	assert.False(rb.IsFull)
}


func TestEnqueue_WhenFull_ShouldReturnError(t *testing.T) {
	assert := assert.New(t)
	rb := createRingBuffer(2)

	_ = rb.Enqueue(&task{id: 1})
	_ = rb.Enqueue(&task{id: 2})

	err := rb.Enqueue(&task{id: 3})

	assert.Error(err)
	assert.EqualError(err, "Can't enqueue, ring buffer is full.")
	assert.Equal(2, rb.CurrentSize)
	assert.True(rb.IsFull)
}


func TestDequeue_WhenEmpty_ShouldReturnNil(t *testing.T) {
	assert := assert.New(t)
	rb := createRingBuffer(2)

	out := rb.Dequeue()

	assert.Nil(out)
	assert.Equal(0, rb.CurrentSize)
}


// ---------------------------------------------------------------------------
// ---------------------------------------------------------------------------
// TESTING TASK (tasks.go)
// ---------------------------------------------------------------------------
// ---------------------------------------------------------------------------
func TestTask_SetStateReady(t *testing.T) {
	assert := assert.New(t)
	tk := &task{}
	tk.SetStateReady()
	assert.Equal(ready, tk.state)
}


func TestTask_SetStateWaiting(t *testing.T) {
	assert := assert.New(t)
	tk := &task{}
	tk.SetStateWaiting()
	assert.Equal(waiting, tk.state)
}


func TestTask_SetStateProcessing(t *testing.T) {
	assert := assert.New(t)
	tk := &task{}
	tk.SetStateProcessing()
	assert.Equal(processing, tk.state)
}


func TestTask_SetAction(t *testing.T) {
	assert := assert.New(t)
	tk := &task{}

	dummyAction := func(params map[string]interface{}) error {
		return nil
	}

	tk.SetAction(dummyAction)
	assert.NotNil(tk.action)
}


func TestTask_SetParams(t *testing.T) {
	assert := assert.New(t)
	tk := &task{}

	params := map[string]interface{}{
		"key": "value",
	}
	tk.SetParams(params)
	assert.Equal(params, tk.params)
}


func TestTask_SetExternalId(t *testing.T) {
	assert := assert.New(t)
	tk := &task{}
	tk.SetExternalId("abc-123")
	assert.Equal("abc-123", tk.externalId)
}


func TestTask_SetId(t *testing.T) {
	assert := assert.New(t)
	tk := &task{}
	tk.SetId(42)
	assert.Equal(42, tk.id)
}


func TestTask_Clean(t *testing.T) {
	assert := assert.New(t)
	tk := &task{
		state:      processing,
		id:         7,
		externalId: "ext-42",
		params:     map[string]interface{}{"foo": "bar"},
		action: func(p map[string]interface{}) error {
			return nil
		},
	}

	tk.Clean()

	assert.Equal(ready, tk.state)
	assert.Nil(tk.params)
	assert.Nil(tk.action)
	assert.Equal("", tk.externalId)
	assert.Equal(7, tk.id, "id should remain unchanged after Clean()")
}


func TestTask_CallAction_NilActionAndParams(t *testing.T) {
	assert := assert.New(t)

	tk := &task{}
	err := tk.CallAction()

	assert.Error(err)
	assert.EqualError(err, "Task action and/or params are nil, cannot make call.")
}


func TestTask_CallAction_NilParams(t *testing.T) {
	assert := assert.New(t)

	tk := &task{
		action: func(params map[string]interface{}) error {
			return nil
		},
	}

	err := tk.CallAction()
	assert.Error(err)
	assert.EqualError(err, "Task action and/or params are nil, cannot make call.")
}


func TestTask_CallAction_NilAction(t *testing.T) {
	assert := assert.New(t)

	tk := &task{
		params: map[string]interface{}{"a": 1},
	}

	err := tk.CallAction()
	assert.Error(err)
	assert.EqualError(err, "Task action and/or params are nil, cannot make call.")
}


func TestTask_CallAction_Success(t *testing.T) {
	assert := assert.New(t)

	called := false
	tk := &task{
		params: map[string]interface{}{"x": 42},
		action: func(params map[string]interface{}) error {
			if params["x"] == 42 {
				called = true
			}
			return nil
		},
	}

	err := tk.CallAction()
	assert.NoError(err)
	assert.True(called, "Expected the action to be called")
}


func TestTask_CallAction_ErrorFromAction(t *testing.T) {
	assert := assert.New(t)

	tk := &task{
		params: map[string]interface{}{},
		action: func(params map[string]interface{}) error {
			return errors.New("fail inside action")
		},
	}

	err := tk.CallAction()
	assert.Error(err)
	assert.EqualError(err, "fail inside action")
}