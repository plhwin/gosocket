package gosocket

import (
	"errors"
	"reflect"
)

type caller struct {
	Func        reflect.Value // registered event handler
	Args        reflect.Type  // the data type of event processing function input args
	ArgsPresent bool          // whether the event processing function has the second input parameter(used to receive $args from client requests ["$event",$args,"$id"])
	Id          reflect.Type  // the id of message, client maintenance
	IdPresent   bool          // whether the event processing function has the third input parameter(used to receive $id from client requests ["$event",$args,"$id"])
	Out         bool          // does the event processing function return a value
}

var (
	ErrorCallerFunc   = errors.New("f is not function")
	ErrorCallerArgs   = errors.New("f should have 1 or 2 or 3 args")
	ErrorCallerReturn = errors.New("f should return not more than one value")
)

// parses function passed by using reflection, and stores its representation
// for further call on message or ack
func newCaller(f interface{}) (*caller, error) {
	fVal := reflect.ValueOf(f)
	// f is not a legal function
	if fVal.Kind() != reflect.Func {
		return nil, ErrorCallerFunc
	}

	fType := fVal.Type()
	// the return value of the event handler function cannot exceed 1 (0 or 1 are possible)
	if fType.NumOut() > 1 {
		return nil, ErrorCallerReturn
	}

	curCaller := &caller{
		Func: fVal,
		Out:  fType.NumOut() == 1,
	}
	if fType.NumIn() == 1 {
		// event only
		curCaller.Args = nil
		curCaller.ArgsPresent = false
		curCaller.Id = nil
		curCaller.IdPresent = false
	} else if fType.NumIn() == 2 {
		// event + args
		curCaller.Args = fType.In(1)
		curCaller.ArgsPresent = true
		curCaller.Id = nil
		curCaller.IdPresent = false
	} else if fType.NumIn() == 3 {
		// event + args + id
		curCaller.Args = fType.In(1)
		curCaller.ArgsPresent = true
		curCaller.Id = fType.In(2)
		curCaller.IdPresent = true
	} else {
		// the input parameters of the event processing function can only be 1 or 2
		return nil, ErrorCallerArgs
	}

	return curCaller, nil
}

// returns function parameter as it is present in it using reflection
func (c *caller) getArgs() interface{} {
	return reflect.New(c.Args).Interface()
}

// calls function with given arguments from its representation using reflection
func (c *caller) callFunc(client interface{}, args interface{}) []reflect.Value {
	//nil is untyped, so use the default empty value of correct type
	if args == nil {
		args = c.getArgs()
	}

	in := []reflect.Value{reflect.ValueOf(client), reflect.ValueOf(args).Elem()}
	if !c.ArgsPresent {
		in = in[0:1]
	}

	return c.Func.Call(in)
}
