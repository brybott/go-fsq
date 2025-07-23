package fsq

import "errors"

type ringBuffer struct {
	MaxSize int
	CurrentSize int
	IsFull bool
	items *[]*task
	head int
	tail int
}


func (rb *ringBuffer) Enqueue(task *task) error {
	if rb.CurrentSize == rb.MaxSize {
		return errors.New("Can't enqueue, ring buffer is full.")
	}

	rb.tail = (rb.head + rb.CurrentSize) % rb.MaxSize
	rb.CurrentSize++
	(*rb.items)[rb.tail] = task

	if rb.CurrentSize == rb.MaxSize {
		rb.IsFull = true
	}

	return nil
}


func (rb *ringBuffer) Dequeue() *task {
	if rb.CurrentSize == 0 {
		return nil
	}

	task := (*rb.items)[rb.head]

	(*rb.items)[rb.head] = nil
	rb.head = (rb.head + 1) % rb.MaxSize
	rb.CurrentSize--
	rb.IsFull = false

	return task
}