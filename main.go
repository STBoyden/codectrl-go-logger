package codectrl

import (
	"context"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	e "github.com/STBoyden/codectrl-go-logger/error"
	h "github.com/STBoyden/codectrl-go-logger/hashbag"

	b "github.com/STBoyden/codectrl-go-protobufs/data/backtrace_data"
	l "github.com/STBoyden/codectrl-go-protobufs/data/log"
	logsService "github.com/STBoyden/codectrl-go-protobufs/logs_service"
	"github.com/go-errors/errors"
	grpc "google.golang.org/grpc"
)

type createLogParams struct {
	surround               uint32
	functionName           string
	functionNameOccurences h.HashBag[string]
}

func createLog(message string, params ...createLogParams) (*l.Log, error) {
	// function_name := ""

	if len(params) > 0 {
		params := params[0]

		// if params.function_name != "" {
		// 	function_name = params.function_name
		// }

		if params.surround == 0 {
			params.surround = 3
		}
	}

	parameters := params[0]

	log := l.Log{
		Uuid:        "",
		Stack:       []*b.BacktraceData{},
		LineNumber:  0,
		FileName:    "",
		CodeSnippet: map[uint32]string{},
		Message:     message,
		MessageType: reflect.TypeOf(message).String(),
		Address:     "",
		Warnings:    []string{},
		Language:    "Go",
	}

	stack, err := getStackTrace(&log)

	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	log.Stack = *stack

	if !(len(log.GetStack()) > 0) {
		return &log, nil
	}

	last := log.GetStack()[len(log.GetStack())-1]
	if last != nil {
		log.LineNumber = last.GetLineNumber()
		log.FileName = last.GetFilePath()
		snippet, err := getCodeSnippet(last.FilePath, &log, parameters.surround, "", nil)

		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		log.CodeSnippet = *snippet
	}

	return &log, nil
}

// Optional parameters for the Logger interface methods.
type LoggerParams struct {
	surround uint32
	host     string
	port     string
}

// Creates a new LoggerParams using the given parameters.
func NewLoggerParams(surround uint32, host string, port string) LoggerParams {
	return LoggerParams{surround: surround, host: host, port: port}
}

// Creates a new, empty LoggerParams.
func NewEmptyLoggerParams() LoggerParams {
	return LoggerParams{}
}

// Main Logger struct
type Logger struct{}

// Creates a new Logger.
func NewLogger() Logger {
	return Logger{}
}

// TODO: Add batch logging

type loggerInterface interface {
	Log(message string, params ...LoggerParams) (*logsService.RequestResult, error)
	LogIf(message string, condition func(params ...struct{}) bool, params ...LoggerParams) (*logsService.RequestResult, error)
	LogWhenEnv(message string, params ...LoggerParams) (*logsService.RequestResult, error)

	log(log l.Log, host string, port string) (*logsService.RequestResult, error)
}

// Main Log function, sends a log whenever this function is called, assuming
// the connection is valid.
func (logger Logger) Log(message string, params ...LoggerParams) (*logsService.RequestResult, error) {
	host := "127.0.0.1"
	port := "3002"
	surround := uint32(3)

	if len(params) > 0 {
		params := params[0]

		if params.host != "" {
			host = params.host
		}

		if params.port != "" {
			port = params.port
		}

		if params.surround != 0 {
			surround = params.surround
		}
	}

	log, err := createLog(message, createLogParams{surround: surround})

	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return logger.log(*log, host, port)
}

// Log function that only connects and sends if the given `condition` function
// pointer resolves to true.
func (logger Logger) LogIf(message string, condition func(params ...struct{}) bool, params ...LoggerParams) (*logsService.RequestResult, error) {
	if !condition() {
		return nil, errors.Wrap("Condition was not true", 0)
	}

	return logger.Log(message, params...)
}

// Log function that only connects and sends when the "CODECTRL_DEBUG"
// environment variable is set.
func (logger Logger) LogWhenEnv(message string, params ...LoggerParams) (*logsService.RequestResult, error) {
	_, present := os.LookupEnv("CODECTRL_DEBUG")

	if !present {
		return nil, errors.Wrap("Environment variable CODECTRL_DEBUG not set", 0)
	}

	return logger.Log(message, params...)
}

func (logger Logger) log(log l.Log, host string, port string) (*logsService.RequestResult, error) {
	connection, err := grpc.Dial(fmt.Sprintf("%s:%s", host, port), grpc.WithInsecure())

	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	client := logsService.NewLogClientClient(connection)

	resultChannel := make(chan logsService.RequestResult)
	errorChannel := make(chan error)

	go func() {
		defer connection.Close()

		r, err := client.SendLog(context.Background(), &log)

		if r != nil {
			resultChannel <- *r
			errorChannel <- nil
		} else if err != nil {
			errorChannel <- err
		}
	}()

	r := <-resultChannel

	return &r, <-errorChannel
}

func deduplicateStack(stack []*b.BacktraceData) []*b.BacktraceData {
	occurred := map[uint32]bool{}
	result := []*b.BacktraceData{}

	for x := range stack {
		if occurred[stack[x].GetLineNumber()] != true {
			occurred[stack[x].GetLineNumber()] = true

			result = append(result, stack[x])
		}
	}

	return result
}

// NOTE: For some reason, this seems to skip some layers when generating the
// stack trace. I am unsure whether this is down to the implementation in
// go-errors, or that the functions are being inlined in a way which makes it
// impossible for the stacktrace to generate for those lines.
func getStackTrace(log *l.Log) (*[]*b.BacktraceData, error) {
	fakeError := errors.Wrap("fake error", 0)
	stack := fakeError.StackFrames()
	bstack := []*b.BacktraceData{}

	for _, frame := range stack {
		switch frame.Package {
		case "runtime", "testing", "github.com/STBoyden/codectrl-go-logger":
			continue
		default:
		}

		if strings.Contains(frame.File, os.Getenv("GOROOT")) {
			continue
		}

		code, err := frame.SourceLine()

		if err != nil {
			codeResult, err := getCode(frame.File, uint32(frame.LineNumber))

			if err != nil {
				return nil, errors.Wrap(err, 0)
			}

			code = *codeResult
		}

		bstack = append(
			[]*b.BacktraceData{
				{
					LineNumber:   uint32(frame.LineNumber),
					ColumnNumber: uint32(0),
					FilePath:     frame.File,
					Name:         frame.Name,
					Code:         code,
				},
			},
			bstack...)
	}

	bstack = deduplicateStack(bstack)

	return &bstack, nil
}

func getCode(filePath string, lineNumber uint32) (*string, error) {
	file, err := os.Open(filePath)

	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	defer file.Close()

	contentBytes, err := io.ReadAll(file)

	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	content := string(contentBytes)
	lines := strings.Split(content, "\n")

	if len(lines) < int(lineNumber) {
		return nil, errors.Wrap(e.New(e.LineNumTooLargeError, "Line number is too large for this file."), 0)
	} else if int(lineNumber) <= 0 {
		return nil, errors.Wrap(e.New(e.LineNumZeroError, "Line number is zero or negative."), 0)
	}

	line := lines[lineNumber-1]

	return &line, nil
}

// TODO: Account for batch logging

func getCodeSnippet(filePath string, log *l.Log, surround uint32, functionName string, functionNameOccurences *h.HashBag[string]) (*map[uint32]string, error) {
	file, err := os.Open(filePath)
	lineNumber := int(log.LineNumber)
	offsetLine := lineNumber - int(surround)

	if offsetLine < 0 {
		offsetLine = 0
	}

	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	defer file.Close()

	contentBytes, err := io.ReadAll(file)

	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	content := string(contentBytes)
	lines := strings.Split(content, "\n")

	snippet := map[uint32]string{}

	for index, line := range lines {
		if index < offsetLine {
			continue
		} else if index > lineNumber+int(surround) {
			break
		}

		snippet[uint32(index+1)] = line
	}

	return &snippet, nil
}
