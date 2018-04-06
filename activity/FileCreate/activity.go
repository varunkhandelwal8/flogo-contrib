package FileCreate

import (
	"github.com/TIBCOSoftware/flogo-lib/core/activity"
	"github.com/TIBCOSoftware/flogo-lib/logger"
	"os"
	//"fmt"
)

// THIS IS ADDED
// log is the default package logger which we'll use to log
var log = logger.GetLogger("activity-FileCreate")

// MyActivity is a stub for your Activity implementation
type MyActivity struct {
	metadata *activity.Metadata
}

// NewActivity creates a new activity
func NewActivity(metadata *activity.Metadata) activity.Activity {
	return &MyActivity{metadata: metadata}
}

// Metadata implements activity.Activity.Metadata
func (a *MyActivity) Metadata() *activity.Metadata {
	return a.metadata
}

// THIS HAS CHANGED
// Eval implements activity.Activity.Eval
func (a *MyActivity) Eval(context activity.Context) (done bool, err error)  {
	// Get the activity data from the context
	path := context.GetInput("path").(string)
	//salutation := context.GetInput("salutation").(string)
	//var _, err1 = os.Stat(path)

	// create file if not exists
	//if os.IsNotExist(_) {
		os.Create(path)
	//if isError(err1) { return }
	//	defer file.Close()
	//}

	//fmt.Println("==> done creating file", path)
	// Use the log object to log the greeting
	log.Debugf("done creating file")

	// Set the result as part of the context
	context.SetOutput("output", "done")

	// Signal to the Flogo engine that the activity is completed
	return true, nil
}

