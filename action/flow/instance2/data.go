package instance2

import (
	"github.com/TIBCOSoftware/flogo-contrib/action/flow/definition"
	"github.com/TIBCOSoftware/flogo-lib/logger"
	"runtime/debug"
	"fmt"
	"github.com/TIBCOSoftware/flogo-lib/core/activity"
	"github.com/TIBCOSoftware/flogo-contrib/action/flow/model"
	"github.com/TIBCOSoftware/flogo-lib/core/data"
	"errors"
	"strings"
	"github.com/TIBCOSoftware/flogo-lib/core/action"
)

func NewTaskData(execEnv *ExecEnv, task *definition.Task) *TaskData {
	var taskData TaskData

	taskData.execEnv = execEnv
	taskData.task = task

	//taskData.TaskID = task.ID

	return &taskData
}

type TaskData struct {
	execEnv *ExecEnv
	task    *definition.Task
	status  int


	workingData    map[string]*data.Attribute

	//state   int
	//done    bool
	//attrs   map[string]*data.Attribute
	//
	inScope  data.Scope
	outScope data.Scope
	//
	//changes int
	//
	//taskID string //needed for serialization
}

// InputScope get the InputScope of the task instance
func (td *TaskData) InputScope() data.Scope {

	if td.inScope != nil {
		return td.inScope
	}

	if len(td.task.ActivityConfig().Ref()) > 0 {

		act := activity.Get(td.task.ActivityConfig().Ref())
		td.inScope = NewFixedTaskScope(act.Metadata().Input, td.task, true)

	} else if td.task.IsScope() {

		//add flow scope
	}

	return td.inScope
}

// OutputScope get the InputScope of the task instance
func (td *TaskData) OutputScope() data.Scope {

	if td.outScope != nil {
		return td.outScope
	}

	if len(td.task.ActivityConfig().Ref()) > 0 {

		act := activity.Get(td.task.ActivityConfig().Ref())
		td.outScope = NewFixedTaskScope(act.Metadata().Output, td.task, false)

		logger.Debugf("OutputScope: %v\n", td.outScope)
	} else if td.task.IsScope() {

		//add flow scope
	}

	return td.outScope
}

/////////////////////////////////////////
// TaskData - activity.Context Implementation

func (td *TaskData) ActionContext() action.Context {
	return td.execEnv.Instance.ActionContext()
}

// TaskName implements activity.Context.TaskName method
func (td *TaskData) TaskName() string {
	return td.task.Name()
}

// GetInput implements activity.Context.GetInput
func (td *TaskData) GetInput(name string) interface{} {

	val, found := td.InputScope().GetAttr(name)
	if found {
		return val.Value()
	}

	return nil
}

// GetOutput implements activity.Context.GetOutput
func (td *TaskData) GetOutput(name string) interface{} {

	val, found := td.OutputScope().GetAttr(name)
	if found {
		return val.Value()
	}

	return nil
}

// SetOutput implements activity.Context.SetOutput
func (td *TaskData) SetOutput(name string, value interface{}) {

	logger.Debugf("SET OUTPUT: %s = %v\n", name, value)
	td.OutputScope().SetAttrValue(name, value)
}

// FlowDetails implements activity.Context.FlowName method
func (td *TaskData) FlowDetails() activity.FlowDetails {
	return nil //td.execEnv.Instance
}

/////////////////////////////////////////
// TaskData - TaskContext Implementation

// Status implements flow.TaskContext.GetState
func (td *TaskData) Status() int {
	return td.status
}

// SetStatus implements flow.TaskContext.SetStatus
func (td *TaskData) SetStatus(status int) {
	td.status = status
	//td.execEnv.Instance.ChangeTracker.trackTaskData(&TaskDataChange{ChgType: CtUpd, ID: td.task.ID(), TaskData: td})
}

func (td *TaskData) HasWorkingData() bool {
	return td.workingData != nil
}

func (td *TaskData) GetSetting(setting string) (value interface{}, exists bool) {

	value, exists = td.task.GetSetting(setting)

	if !exists {
		return nil, false
	}

	strValue, ok := value.(string)

	if ok && strValue[0] == '$' {

		v, err := definition.GetDataResolver().Resolve(strValue, td.execEnv)
		if err != nil {
			return nil, false
		}

		return v, true

	} else {
		return value, true
	}
}

func (td *TaskData) AddWorkingData(attr *data.Attribute) {

	if td.workingData == nil {
		td.workingData = make(map[string]*data.Attribute)
	}
	td.workingData[attr.Name()] = attr
}


func (td *TaskData) UpdateWorkingData(key string, value interface{}) error {

	if td.workingData == nil {
		return errors.New("working data '" + key + "' not defined")
	}

	attr, ok := td.workingData[key]

	if ok {
		attr.SetValue(value)
	} else {
		return errors.New("working data '" + key + "' not defined")
	}

	return nil
}

func (td *TaskData) GetWorkingData(key string) (*data.Attribute, bool) {
	if td.workingData == nil {
		return nil, false
	}

	v, ok := td.workingData[key]
	return v, ok
}

// Task implements model.TaskContext.Task, by returning the Task associated with this
// TaskData object
func (td *TaskData) Task() *definition.Task {
	return td.task
}

// FromInstLinks implements model.TaskContext.FromInstLinks
func (td *TaskData) FromInstLinks() []model.LinkInst {

	logger.Debugf("FromInstLinks: task=%v\n", td.Task)

	links := td.task.FromLinks()

	numLinks := len(links)

	if numLinks > 0 {
		linkCtxs := make([]model.LinkInst, numLinks)

		for i, link := range links {
			linkCtxs[i], _ = td.execEnv.FindOrCreateLinkData(link)
		}
		return linkCtxs
	}

	return nil
}

// ToInstLinks implements model.TaskContext.ToInstLinks,
func (td *TaskData) ToInstLinks() []model.LinkInst {

	logger.Debugf("ToInstLinks: task=%v\n", td.Task)

	links := td.task.ToLinks()

	numLinks := len(links)

	if numLinks > 0 {
		linkCtxs := make([]model.LinkInst, numLinks)

		for i, link := range links {
			linkCtxs[i], _ = td.execEnv.FindOrCreateLinkData(link)
		}
		return linkCtxs
	}

	return nil
}

// EvalLink implements activity.ActivityContext.EvalLink method
func (td *TaskData) EvalLink(link *definition.Link) (result bool, err error) {

	logger.Debugf("TaskContext.EvalLink: %d\n", link.ID())

	defer func() {
		if r := recover(); r != nil {
			logger.Warnf("Unhandled Error evaluating link '%s' : %v\n", link.ID(), r)

			// todo: useful for debugging
			logger.Debugf("StackTrace: %s", debug.Stack())

			if err != nil {
				err = fmt.Errorf("%v", r)
			}
		}
	}()

	mgr := td.execEnv.flowDef.GetLinkExprManager()

	if mgr != nil {
		result, err = mgr.EvalLinkExpr(link, td.execEnv)
		return result, err
	}

	return true, nil
}

// HasActivity implements activity.ActivityContext.HasActivity method
func (td *TaskData) HasActivity() bool {
	return activity.Get(td.task.ActivityConfig().Ref()) != nil
}

// EvalActivity implements activity.ActivityContext.EvalActivity method
func (td *TaskData) EvalActivity() (done bool, evalErr error) {

	defer func() {
		if r := recover(); r != nil {
			logger.Warnf("Unhandled Error executing activity '%s'[%s] : %v\n", td.task.Name(), td.task.ActivityConfig().Ref(), r)

			// todo: useful for debugging
			logger.Debugf("StackTrace: %s", debug.Stack())

			if evalErr == nil {
				evalErr = NewActivityEvalError(td.task.Name(), "unhandled", fmt.Sprintf("%v", r))
				done = false
			}
		}
		if evalErr != nil {
			logger.Errorf("Execution failed for Activity[%s] in Flow[%s] - %s", td.task.Name(), td.execEnv.flowDef.Name(), evalErr.Error())
		}
	}()

	eval := true

	if td.task.ActivityConfig().InputMapper() != nil {

		err := applyInputMapper(td)

		if err != nil {

			evalErr = NewActivityEvalError(td.task.Name(), "mapper", err.Error())
			return false, evalErr
		}

		eval = applyInputInterceptor(td)
	}

	if eval {

		act := activity.Get(td.task.ActivityConfig().Ref())
		done, evalErr = act.Eval(td)

		if evalErr != nil {
			e, ok := evalErr.(*activity.Error)
			if ok {
				e.SetActivityName(td.task.Name())
			}

			return false, evalErr
		}
	} else {
		done = true
	}

	if done {

		if td.task.ActivityConfig().OutputMapper() != nil {
			applyOutputInterceptor(td)

			appliedMapper, err := applyOutputMapper(td)

			if err != nil {
				evalErr = NewActivityEvalError(td.task.Name(), "mapper", err.Error())
				return done, evalErr
			}

			if !appliedMapper && !td.task.IsScope() {

				logger.Debug("Mapper not applied")
			}
		}
	}

	return done, nil
}

// Failed marks the Activity as failed
func (td *TaskData) Failed(err error) {

	errorMsgAttr := "[A" + td.task.ID() + "._errorMsg]"
	td.execEnv.AddAttr(errorMsgAttr, data.STRING, err.Error())
	errorMsgAttr2 := "[activity." + td.task.ID() + "._errorMsg]"
	td.execEnv.AddAttr(errorMsgAttr2, data.STRING, err.Error())
}


// LinkData represents data associated with an instance of a Link
type LinkData struct {
	execEnv *ExecEnv
	link    *definition.Link
	status  int

	changes int

	linkID int //needed for serialization
}

// NewLinkData creates a LinkData for the specified link in the specified task
// environment
func NewLinkData(execEnv *ExecEnv, link *definition.Link) *LinkData {
	var linkData LinkData

	linkData.execEnv = execEnv
	linkData.link = link

	return &linkData
}

// Status returns the current state indicator for the LinkData
func (ld *LinkData) Status() int {
	return ld.status
}

// SetStatus sets the current state indicator for the LinkData
func (ld *LinkData) SetStatus(status int) {
	ld.status = status
	//ld.execEnv.Instance.ChangeTracker.trackLinkData(&LinkDataChange{ChgType: CtUpd, ID: ld.link.ID(), LinkData: ld})
}

// Link returns the Link associated with ld context
func (ld *LinkData) Link() *definition.Link {
	return ld.link
}

// WorkingDataScope is scope restricted by the set of reference attrs and backed by the specified Task
type WorkingDataScope struct {
	parent      data.Scope
	workingData map[string]*data.Attribute
}

// NewFixedTaskScope creates a FixedTaskScope
func NewWorkingDataScope(parentScope data.Scope, workingData map[string]*data.Attribute) data.Scope {

	scope := &WorkingDataScope{
		parent:      parentScope,
		workingData: workingData,
	}

	return scope
}

// GetAttr implements Scope.GetAttr
func (s *WorkingDataScope) GetAttr(attrName string) (attr *data.Attribute, exists bool) {

	if strings.HasPrefix(attrName, "$current.") {
		val, ok := s.workingData[attrName[9:]]
		if ok {
			return val, true
			//attr, _ = data.NewAttribute(attrName[6:], data.ANY, val)
			//return attr, true
		}
		return nil, false
	} else {
		return s.parent.GetAttr(attrName)
	}
}

// SetAttrValue implements Scope.SetAttrValue
func (s *WorkingDataScope) SetAttrValue(attrName string, value interface{}) error {
	return s.parent.SetAttrValue(attrName, value)
}

// FixedTaskScope is scope restricted by the set of reference attrs and backed by the specified Task
type FixedTaskScope struct {
	attrs    map[string]*data.Attribute
	refAttrs map[string]*data.Attribute
	task     *definition.Task
	isInput  bool
}

// NewFixedTaskScope creates a FixedTaskScope
func NewFixedTaskScope(refAttrs map[string]*data.Attribute, task *definition.Task, isInput bool) data.Scope {

	scope := &FixedTaskScope{
		refAttrs: refAttrs,
		task:     task,
		isInput:  isInput,
	}

	return scope
}

// GetAttr implements Scope.GetAttr
func (s *FixedTaskScope) GetAttr(attrName string) (attr *data.Attribute, exists bool) {

	if len(s.attrs) > 0 {

		attr, found := s.attrs[attrName]

		if found {
			return attr, true
		}
	}

	if s.task != nil {

		var attr *data.Attribute
		var found bool

		if s.isInput {
			attr, found = s.task.ActivityConfig().GetInputAttr(attrName)
		} else {
			attr, found = s.task.ActivityConfig().GetOutputAttr(attrName)
		}

		if !found {
			attr, found = s.refAttrs[attrName]
		}

		return attr, found
	}

	return nil, false
}

// SetAttrValue implements Scope.SetAttrValue
func (s *FixedTaskScope) SetAttrValue(attrName string, value interface{}) error {

	if len(s.attrs) == 0 {
		s.attrs = make(map[string]*data.Attribute)
	}

	logger.Debugf("SetAttr: %s = %v\n", attrName, value)

	attr, found := s.attrs[attrName]

	var err error
	if found {
		err = attr.SetValue(value)
	} else {
		// look up reference for type
		attr, found = s.refAttrs[attrName]
		if found {
			s.attrs[attrName], err = data.NewAttribute(attrName, attr.Type(), value)
		} else {
			logger.Debugf("SetAttr: Attr %s ref not found\n", attrName)
			logger.Debugf("SetAttr: refs %v\n", s.refAttrs)
		}
		//todo: else error
	}

	return err
}


func NewActivityEvalError(taskName string, errorType string, errorText string) *ActivityEvalError {
	return &ActivityEvalError{taskName: taskName, errType: errorType, errText: errorText}
}

type ActivityEvalError struct {
	taskName string
	errType  string
	errText  string
}

func (e *ActivityEvalError) TaskName() string {
	return e.taskName
}

func (e *ActivityEvalError) Type() string {
	return e.errType
}

func (e *ActivityEvalError) Error() string {
	return e.errText
}