package fsq

import "errors"

// valid task state values
const ready string = "r"
const waiting string = "w"
const processing string = "p"

type task struct {
	state string
	action func(params map[string]interface{}) error
	params map[string]interface{}  //should be passed to the "action" func
	id int  //a non mutable (by convention) id that is constant as tasks are re-used
	externalId string  //a mutable "id" that allows users of this package to give an id to the task. The idea is to prevent duplication of tasks, such that a task will not be created if it shares the same id with a task that has a waiting state.
}


func (t *task) Clean() {
	t.SetStateReady()
	t.SetAction(nil)
	t.SetParams(nil)
	t.SetExternalId("")
}


func (t *task) SetStateReady() {
	t.state = ready
}


func (t *task) SetStateWaiting() {
	t.state = waiting
}


func (t *task) SetStateProcessing() {
	t.state = processing
}


func (t *task) CallAction() error {
	if t.action == nil || t.params == nil {
		// Don't expect this to happen, adding for safety.
		return errors.New("Task action and/or params are nil, cannot make call.")
	}
	return t.action(t.params)
}


func (t *task) SetAction(action func(params map[string]interface{}) error) {
	t.action = action
}


func (t *task) SetParams(params map[string]interface{}) {
	t.params = params
}


func (t *task) SetExternalId(externalId string) {
	t.externalId = externalId
}


func (t *task) SetId(id int) {
	t.id = id
}