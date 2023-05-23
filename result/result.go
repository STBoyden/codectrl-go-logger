package result

type ErrorType uint64

const (
	NoError ErrorType = iota
	GrpcError
	LoggerError
	IoError
	LineNumZeroError
	LineNumTooLargeError
	Other
)

type Error struct {
	Type    ErrorType
	Message string
}

type ResultInterface[T any] interface {
	Ok() *T
	IsOk() bool
	Err() *Error
	IsErr() bool
}

type Result[T any] struct {
	ok  *T
	err *Error
}

func (r Result[T]) Ok() *T {
	if r.ok == nil {
		panic("Result was not OK")
	}

	return r.ok
}

func (r Result[T]) Err() *Error {
	if r.err == nil {
		panic("Result was not an error")
	}

	return r.err
}

func (r Result[T]) IsOk() bool {
	return r.ok != nil
}

func (r Result[T]) IsErr() bool {
	return r.err != nil
}

func NewOk[T any](ok *T) Result[T] {
	return Result[T]{ok: ok, err: nil}
}

func NewErr[T any](errorType ErrorType, errorMessage string) Result[T] {
	err := &Error{}

	err.Type = errorType
	err.Message = errorMessage

	return Result[T]{ok: nil, err: err}
}

func New[T any](ok *T, err *Error) Result[T] {
	return Result[T]{ok: ok, err: err}
}

func NewWithTypeAndMessage[T any](ok *T, errType ErrorType, errorMessage string) Result[T] {
	err := &Error{}

	if errType != NoError && errorMessage != "" {
		err.Type = errType
		err.Message = errorMessage
	} else {
		err = nil
	}

	return Result[T]{ok: ok, err: err}
}
