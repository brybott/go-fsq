# go_fsq: A fixed sized queue that runs "tasks"
**Disclaimer: go_fsq is not actively maintained. It is a project for learning. It is not tested in production. No guarantees are implied.**
- fsq or "fixed size queue" is a queue of tasks. Tasks are functions that are to be run concurrently. The max number of items allowed in the queue, and max number of concurrent tasks are determined at the time which the fsq is created.

- Internally, fsq uses a ring buffer for O(1) adding and removing to the queue.

- Additionally, tasks are re-used (when possible) to remove the overhead of creating new tasks in memory for each item added to the queue. The maximum number of tasks that can be created is equal to the maximum size of the queue.

- To prevent duplication, new tasks cannot be added with the same id as tasks that are waiting in the queue. However, there is no logic to prevent duplicating a task that has already been removed from the queue (processed).

- IMPORTANT: Adding to the queue is a fire and forget operation. There is no feedback regarding if a task has been completed successfully or not.

- go_fsc is licensed under the GNU LGPLv3 license.

## How to use
- Run `go get -u github.com/brybott/go_fsq`
- Import the package `import github.com/brybott/go_fsq`
- Initialize the queue
```
maxSize := 100
maxProcesses := 10
name := "my_favorite_queue"
queue := fsc.Init(maxSize, name, maxProcesses)
```
- Add a task to the queue
```
// the function passed as the first parameter to .Add() must have the signature below
// (it must take a single map parameter with string keys and any type of values,  
// and it must return an error/nil).

doStuff := func(params map[string]interface{}) error {
  return nil
}

// params must exist, even if empty.  
// Tasks created with params passed as nil will not be run.
// Don't forget to type convert the values of params as necessary in your function.
params := map[string]interface{}{
    "key": "value",
}

// IDs should not be reused. A task will not be added to the queue if it shares  
the same ID as a task waiting in the queue.
taskId := "stuff-12345"

// Tasks added to the queue are run concurrently, they do not block.  
// The queue will automatically run tasks as processes are available, based on the max number of  
// processes that the queue was created with.
queue.Add(doStuff, params, taskId)
```

## Questions?
Feel free to open an issue, though I can't guarantee that it will be seen :)
