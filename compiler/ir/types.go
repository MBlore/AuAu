package ir

type IRType int

const (
	IRTypeInvalid IRType = iota
	IRTypeI64
	IRTypeU64
)

type IRProgram struct {
}
