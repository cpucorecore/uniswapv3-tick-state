package main

type Input[T any] interface {
	PutOutput(T)
	FinOutput()
}

type Output[T any] interface {
	PutInput(T)
	FinInput()
}

type OutputMountable[T any] interface {
	MountOutput(Output[T])
}

type InputMountable[T any] interface {
	MountInput(Input[T])
}
