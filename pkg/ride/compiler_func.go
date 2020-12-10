package ride

import "fmt"

type arguments []string

type Deferreds interface {
	Add(Deferred, uniqueid, string)
}

type dd struct {
	deferred Deferred
	uniq     uniqueid
	debug    string
}

type deferreds struct {
	name string
	d    []dd
}

func (a *deferreds) Add(deferred2 Deferred, n uniqueid, debug string) {
	a.d = append(a.d, dd{
		deferred: deferred2,
		uniq:     n,
		debug:    debug,
	})
}

func (a *deferreds) Get() []dd {
	return a.d
}

type FuncState struct {
	params
	prev        Fsm
	name        string
	args        arguments
	n           uniqueid
	invokeParam string

	// References that defined inside function.
	deferred []Deferred
	defers   *deferreds
	//exe      Fsm
}

func (a FuncState) retAssigment(as Fsm) Fsm {
	a.deferred = append(a.deferred, as.(Deferred))
	return a
}

func (a FuncState) Property(name string) Fsm {
	panic("FuncState Property")
}

func funcTransition(prev Fsm, params params, name string, args []string, invokeParam string) Fsm {
	// save reference to global scope, where code lower that function will be able to use it.
	n := params.u.next()
	params.r.set(name, n)
	// all variable we add only visible to current scope,
	// avoid corrupting parent state.
	params.r = newReferences(params.r)

	// Function call: verifier or not.
	if invokeParam != "" {
		args = append([]string{invokeParam}, args...)
	}
	for i := range args {
		e := params.u.next()
		//assigments = append(assigments, e)
		params.r.set(args[i], e)
		// set to global
		//globalScope.set(fmt.Sprintf("%s$%d", name, i), e)
	}
	//if invokeParam != "" {
	//	assigments = assigments[1:]
	//}

	return &FuncState{
		prev:        prev,
		name:        name,
		args:        args,
		params:      params,
		n:           n,
		invokeParam: invokeParam,
		defers: &deferreds{
			name: "func " + name,
		},
	}
}

func (a FuncState) Assigment(name string) Fsm {
	n := a.params.u.next()
	//a.assigments = append(a.assigments, n)
	return assigmentFsmTransition(a, a.params, name, n, a.defers)
}

func (a FuncState) Return() Fsm {
	/*
		funcID := a.params.u.next()
		a.globalScope.set(a.name, funcID)
		a.params.c.set(funcID, nil, nil, a.lastStmtOffset, false, a.name)
		// TODO clean args

		// Clean internal assigments.
		for i := len(a.assigments) - 1; i >= 0; i-- {
			a.b.writeByte(OpClearCache)
			a.b.write(encode(a.assigments[i].n))
		}

		a.b.ret()

		// if function has invoke param, it means no other code will be provided.
		if a.invokeParam != "" {
			a.b.startPos()
			for i := len(a.args) - 1; i >= 0; i-- {
				a.b.writeByte(OpCache)
				uniq, ok := a.params.r.get(a.args[i])
				if !ok {
					panic("function param `" + a.args[i] + "` not found")
				}
				a.b.write(encode(uniq))
				a.b.writeByte(OpPop)
			}
			a.b.writeByte(OpCall)
			a.b.write(encode(a.lastStmtOffset))
		}


	*/
	return a.prev.retAssigment(a) //.retAssigment(a.startedAt, a.b.len())
}

func (a FuncState) Long(value int64) Fsm {
	a.deferred = append(a.deferred, a.constant(rideInt(value)))
	return a
}

func (a FuncState) Call(name string, argc uint16) Fsm {
	return callTransition(a, a.params, name, argc, a.defers)
}

func (a FuncState) Reference(name string) Fsm {
	a.deferred = append(a.deferred, reference(a, a.params, name))
	return a
}

func (a FuncState) Boolean(value bool) Fsm {
	a.deferred = append(a.deferred, a.constant(rideBoolean(value)))
	return a
}

func (a FuncState) String(s string) Fsm {
	//a.lastStmtOffset = a.b.len()
	//return constant(a, a.params, rideString(s))
	panic("a")
}

func (a FuncState) Condition() Fsm {
	//a.lastStmtOffset = a.b.len()
	return conditionalTransition(a, a.params, a.defers)
}

func (a FuncState) TrueBranch() Fsm {
	panic("Illegal call `TrueBranch` on `FuncState`")
}

func (a FuncState) FalseBranch() Fsm {
	panic("Illegal call `FalseBranch` on `FuncState`")
}

func (a FuncState) Bytes(b []byte) Fsm {
	//a.lastStmtOffset = a.b.len()
	//return constant(a, a.params, rideBytes(b))
	panic("a")
}

func (a FuncState) Func(name string, args []string, _ string) Fsm {
	panic("Illegal call `Func` is `FuncState`")
}

func (a FuncState) Clean() {

}

func (a FuncState) Write(_ params) {
	pos := a.b.len()
	a.params.c.set(a.n, nil, nil, pos, false, fmt.Sprintf("function %s", a.name))
	//writeDeferred(a.params, a.deferred)
	if len(a.deferred) != 1 {
		panic("len(a.deferred) != 1")
	}
	a.deferred[0].Write(a.params)

	// End of function body. Clear and write assigments.
	for _, v := range a.defers.Get() {
		v.deferred.Clean()
	}
	a.b.ret()

	for _, v := range a.defers.Get() {
		pos := a.b.len()
		a.c.set(v.uniq, nil, nil, pos, false, v.debug)
		v.deferred.Write(a.params)
	}

}