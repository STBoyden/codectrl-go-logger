package error

type ErrorType int

const (
	LoggerError ErrorType = iota
	IoError
	LineNumZeroError
	LineNumTooLargeError
)

type Error struct {
	Message string
	Type    ErrorType
}

func (e *Error) Error() string {
	return e.Message
}

func (e *Error) Unwrap() error { return e }

func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)

	if !ok {
		return false
	}

	return (e.Type == t.Type) && (e.Message == t.Message)
}

func New(errorType ErrorType, reason string) *Error {
	error := &Error{}

	switch errorType {
	case LoggerError:
		error.Message = "codectrl-go-logger: Logger error: " + reason
	case IoError:
		error.Message = "codectrl-go-logger: IO error: " + reason
	case LineNumTooLargeError:
		error.Message = "codectrl-go-logger: Line number too large: " + reason
	case LineNumZeroError:
		error.Message = "codectrl-go-logger: Line number is zero or negative: " + reason
	default:
		error.Message = "codectrl-go-logger: Unknown error: " + reason
	}

	error.Type = errorType

	return error
}
