package diwrapper

import (
	"fmt"
	"os"
	"reflect"

	"github.com/facebookgo/inject"
)

type Initializer interface {
	Init() error
}

// Cleaners are called when everything ends, note that Stop() must be called explicitly.
type Cleaner interface {
	Clean() error
}

type InjectWrapper struct {
	g *inject.Graph
	// this slice is here because we want to initialize objects in the order as they are added (after the graph is generated):
	objects    [][]*inject.Object
	tmpObjects []*inject.Object
}

// NewDebug starts a diwrapper with debug output
func NewDebug() *InjectWrapper {
	di := New()
	di.g.Logger = &log{}
	return di
}

func New() *InjectWrapper {
	var g inject.Graph
	return &InjectWrapper{
		g:          &g,
		objects:    [][]*inject.Object{},
		tmpObjects: []*inject.Object{},
	}
}

func (i *InjectWrapper) log(format string, v ...interface{}) {
	if i.g.Logger != nil {
		i.g.Logger.Debugf(format, v...)
	}
}

func (i *InjectWrapper) WithObjects(objects ...interface{}) *InjectWrapper {
	for _, obj := range objects {
		i.WithObject(obj)
	}
	return i
}

func (i *InjectWrapper) WithObject(object interface{}) *InjectWrapper {
	i.log("Adding %T", object)
	o := &inject.Object{Value: object}
	if err := i.g.Provide(o); err != nil {
		panic(fmt.Sprintf("Error providing object %T:%s", object, err.Error()))
	}
	i.tmpObjects = append(i.tmpObjects, o)
	return i
}

// WithObjectOrErr is a helper methods to be used with initializers which return a pointer and error
func (i *InjectWrapper) WithObjectOrErr(object interface{}, err error) *InjectWrapper {
	if err != nil {
		panic(err)
	}
	i.log("Adding %T", object)
	o := &inject.Object{Value: object}
	if err := i.g.Provide(o); err != nil {
		panic(fmt.Sprintf("Error providing object %T:%s", object, err.Error()))
	}
	i.tmpObjects = append(i.tmpObjects, o)
	return i
}

func (i *InjectWrapper) WithNamedObject(name string, obj interface{}) *InjectWrapper {
	i.log("Adding %s: %T", name, obj)
	o := &inject.Object{Name: name, Value: obj}
	if err := i.g.Provide(o); err != nil {
		panic(fmt.Sprintf("Error providing named object %s.%T:%s", name, obj, err.Error()))
	}
	i.tmpObjects = append(i.tmpObjects, o)
	return i
}

func (i *InjectWrapper) InitAsync() *InjectWrapper {
	i.objects = append(i.objects, i.tmpObjects)
	i.tmpObjects = []*inject.Object{}
	return i
}

func (i *InjectWrapper) InitSync() *InjectWrapper {
	for _, obj := range i.tmpObjects {
		i.objects = append(i.objects, []*inject.Object{obj})
	}
	i.tmpObjects = []*inject.Object{}
	return i
}

func (i *InjectWrapper) AllObjects() []interface{} {
	//if len(i.g.Objects()) != len(i.objects) { panic(fmt.Sprintf("Invalid objects size: %d!=%d", len(i.g.Objects()), len(i.objects))) }
	res := []interface{}{}
	for _, objs := range i.objects {
		for _, o := range objs {
			res = append(res, o.Value)
		}
	}
	return res
}

// MustFindObject privides an object of the specified type and name (name can be empty for unnamed objects). Note that
// this function is only for debugging and testing. In production, objects should be used injected and never retrieved
// with this. That's why this method panics!
func (i InjectWrapper) MustGetNamedObject(sample interface{}, name string) interface{} {
	sampleType := reflect.TypeOf(sample)
	if sampleType.Kind() != reflect.Ptr {
		panic(fmt.Sprintf("Sample must be interface, found %T", sample))
	}
	for _, objs := range i.objects {
		for _, obj := range objs {
			if reflect.TypeOf(obj.Value) == sampleType && obj.Name == name {
				return obj.Value
			}
		}
	}
	panic(fmt.Sprintf("Object not found: %s.%T", name, sample))
}

// MustGetObject: see MustGetNamedObject
func (i InjectWrapper) MustGetObject(sample interface{}) interface{} {
	return i.MustGetNamedObject(sample, "")
}

func (i *InjectWrapper) CheckNoImplicitObjects() *InjectWrapper {
	for _, o := range i.g.Objects() {
		var oOK bool
		for _, objs := range i.objects {
			for _, obj := range objs {
				if obj.Value == o.Value {
					oOK = true
				}
			}
		}
		if oOK {
			i.log("%T OK\n", o.Value)
		} else {
			panic(fmt.Sprintf("%T not explicitly created", o.Value))
		}
	}

	return i
}

// InitializeGraphWithImplicitObjects initializes a graph allowing implicitly created objects. Those are objects not specified with one of the With...() methods.
func (i *InjectWrapper) InitializeGraphWithImplicitObjects() *InjectWrapper {
	i.InitSync()
	i.log("Initializing %d objects", len(i.objects))

	if err := i.g.Populate(); err != nil {
		panic(fmt.Sprintf("Error populating graph: %s", err))
	}
	for _, obj := range i.AllObjects() {
		if initializer, is := obj.(Initializer); is {
			i.log("Initializing %T", obj)
			if err := initializer.Init(); err != nil {
				panic(fmt.Sprintf("Error initializing privided object %T:%s", obj, err.Error()))
			}
			i.log("Initialized %T", obj)
		}
	}

	return i
}

// InitializeGraph initializes a graph, but fails if an object is not specified with one of the With() methods.
func (i *InjectWrapper) InitializeGraph() *InjectWrapper {
	_ = i.InitializeGraphWithImplicitObjects()
	return i.CheckNoImplicitObjects()
}

func (i *InjectWrapper) Stop() {
	for _, obj := range i.AllObjects() {
		if cleaner, is := obj.(Cleaner); is {
			i.log("Cleaning %T", obj)
			if err := cleaner.Clean(); err != nil {
				fmt.Fprintf(os.Stderr, "Error cleaning %T: %+v\n", obj, err)
			}
		}
	}
}

func (i *InjectWrapper) Stopper() func() {
	return func() {
		i.Stop()
	}
}

type log struct{}

func (l *log) Debugf(format string, v ...interface{}) {
	fmt.Printf(format+"\n", v...)
}
