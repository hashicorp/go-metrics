package logger

import (
	log "github.com/hashicorp/go-hclog"
)

// OurLogger implements an empty logger struct to wrap the go-hclog logger
type OurLogger struct {
}

//Logger interface is used to make stuff happen with things in places
type Logger interface {
	Error(string, ...interface{})
}

// Error wraps the go-hclog Error function and logs a separate error for each argument in the error object passed
func Error(msg string, l Logger, args ...interface{}) {
	l.Error(msg, "msg", args)
	// I thought I would try annotating & commenting out codethat didn't work for me in for the initial
	// code review instead of deleting and re-writing so much that I end up telling you I have interfaces
	// when I've deleted them and only have structs. (or instance)
	// Currently playing with visual flags to identify blocks as items that will be discarded and are there to show my thinking
	// or prep a question. Just workshopping this so let me know if it makes CR better/worse! And definitely don't feel obliged
	// to give a long answer (or any answer) just becuase I wrote a lot in a section, just trying to figure out this async comms life!

	// *************    		[DISCARDED ATTEMPT]				*************
	// fmt.Printf("%v", args)
	// fmt.Println("---")
	// for _, v := range args {
	// 	l.Error(msg, "msg", v)
	}

	// fmt.Printf("%v", v)
	// fmt.Println("---")

	// I sat with this a while because I was uncomfortable throwing in a single placeholder string
	// when the args interface could hold a variable number of members.
	// I was confused because just looping with `l.Error(msg, a)` still produced `EXTRA_VALUE_AT_END `
	// Looked at the go-hclog repo a bit more and  I think I get the structure now.  My remaining question
	// is why do I get seemingly identical behavior from both looping and non-looping implementation.

	// *************    		[QUESTION]				*************
	// I think the underlying "Go" question there is about looping over interfaces. I'm not sure why I can range over args
	// but was not able to do a nested loop and range over `v` got the error--- `cannot range over v (type interface {})`
	// First thought is that args' underlying type is an object and the underlying type for v was, in this case, *net.OpError.
	// If that's so, is the "right" thing to do some sort of TypeOf/reflect statement and a switch for different circumstances?
	// Or is there something more interface-first/Go-like I should use if I want the method to handle a number of different
	// possible input values

	// When I came back to this it felt pretty clear that line 18 is actually fine
	// for this case because the go-hclog `msg=`seems to capture the whole err object as output.

}
func (ol *OurLogger) Error(msg string, args ...interface{}) {
	ol.Error(msg, args)
}

//New creates and returns a new go-hclog logger
func (ol *OurLogger) New() Logger {
	l := log.Default()
	return l
}
