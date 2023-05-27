package codectrl

import (
	"context"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/Authentura/codectrl-go-logger/hashbag"
	"github.com/Authentura/codectrl-go-logger/result"

	b "github.com/Authentura/codectrl-go-protobufs/data/backtrace_data"
	l "github.com/Authentura/codectrl-go-protobufs/data/log"
	logsService "github.com/Authentura/codectrl-go-protobufs/logs_service"
	"github.com/go-errors/errors"
	grpc "google.golang.org/grpc"
)

type createLogParams struct {
	surround               uint32
	functionName           string
	functionNameOccurences hashbag.HashBag[string]
}

func createLog(message string, params ...createLogParams) l.Log {
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

	getStackTrace(&log)

	if last := log.GetStack()[len(log.GetStack())-1]; last != nil {
		log.LineNumber = last.GetLineNumber()
		log.FileName = last.GetFilePath()
	}

	return log
}

type LoggerParams struct {
	surround uint32
	host     string
	port     string
}

type Logger struct{}

// TODO: Add Log variants.
// - [x] Log
// - [ ] LogIf
// - [ ] LogWhenEnv
//
// TODO: Add batch logging

type loggerInterface interface {
	Log(message string, params ...LoggerParams) result.Result[chan logsService.RequestResult]
	log(log l.Log, host string, port string) result.Result[chan logsService.RequestResult]
}

func (logger Logger) Log(message string, params ...LoggerParams) result.Result[logsService.RequestResult] {
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

	log := createLog(message, createLogParams{surround: surround})

	return logger.log(log, host, port)
}

func (logger Logger) log(log l.Log, host string, port string) result.Result[logsService.RequestResult] {
	connection, err := grpc.Dial(fmt.Sprintf("%s:%s", host, port), grpc.WithInsecure())

	if err != nil {
		return result.NewErr[logsService.RequestResult](result.GrpcError, err.Error())
	}

	client := logsService.NewLogClientClient(connection)

	resultChannel := make(chan logsService.RequestResult)
	errorChannel := make(chan result.Error)

	go func() {
		defer connection.Close()

		_result, err := client.SendLog(context.Background(), &log)

		if _result != nil {
			resultChannel <- *_result
			errorChannel <- result.Error{Type: result.NoError}
		} else if err != nil {
			errorChannel <- result.Error{Type: result.GrpcError, Message: err.Error()}
		}
	}()

	_result := <-resultChannel
	error := <-errorChannel

	return result.New(&_result, &error)
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
func getStackTrace(log *l.Log) {
	fakeError := errors.New("")
	stack := fakeError.StackFrames()

	for _, frame := range stack {
		if strings.Contains(frame.File, os.Getenv("GOROOT")) || (strings.Contains(frame.File, "codectrl-go") && !strings.Contains(frame.File, "example")) {
			continue
		}

		code, err := frame.SourceLine()

		if err != nil {
			codeResult := getCode(frame.File, uint32(frame.LineNumber))
			if codeResult.IsOk() {
				code = *codeResult.Ok()
			}
		}

		log.Stack = append(
			[]*b.BacktraceData{
				{
					LineNumber:   uint32(frame.LineNumber),
					ColumnNumber: uint32(0),
					FilePath:     frame.File,
					Name:         frame.Name,
					Code:         code,
				},
			},
			log.Stack...)
	}

	log.Stack = deduplicateStack(log.Stack)
}

func getCode(filePath string, lineNumber uint32) result.Result[string] {
	file, err := os.Open(filePath)

	if err != nil {
		return result.NewErr[string](result.IoError, err.Error())
	}

	defer file.Close()

	contentBytes, err := io.ReadAll(file)

	if err != nil {
		return result.NewErr[string](result.IoError, err.Error())
	}

	content := string(contentBytes)
	lines := strings.Split(content, "\n")

	if len(lines) < int(lineNumber) {
		return result.NewErr[string](result.LineNumTooLargeError, "Line number is too large for this file.")
	} else if int(lineNumber) <= 0 {
		return result.NewErr[string](result.LineNumZeroError, "Line number is zero or negative.")
	}

	line := lines[lineNumber-1]

	return result.NewOk(&line)
}

// TODO: Add code snippet retrieval function
