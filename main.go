package codectrl

import (
	"context"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	b "github.com/Authentura/codectrl-go-protobufs/data/backtrace_data"
	l "github.com/Authentura/codectrl-go-protobufs/data/log"
	logsService "github.com/Authentura/codectrl-go-protobufs/logs_service"
	"github.com/go-errors/errors"
	grpc "google.golang.org/grpc"
)

type createLogParams struct {
	surround               uint32
	functionName           string
	functionNameOccurences hashbag[string]
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
	Log(message string, params ...LoggerParams) Result[chan logsService.RequestResult]
	log(log l.Log, host string, port string) Result[chan logsService.RequestResult]
}

func (logger Logger) Log(message string, params ...LoggerParams) Result[logsService.RequestResult] {
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

func (logger Logger) log(log l.Log, host string, port string) Result[logsService.RequestResult] {
	connection, err := grpc.Dial(fmt.Sprintf("%s:%s", host, port), grpc.WithInsecure())

	if err != nil {
		return NewErrResult[logsService.RequestResult](GrpcError, err.Error())
	}

	client := logsService.NewLogClientClient(connection)

	resultChannel := make(chan logsService.RequestResult)
	errorChannel := make(chan Error)

	go func() {
		defer connection.Close()

		result, err := client.SendLog(context.Background(), &log)

		if result != nil {
			resultChannel <- *result
			errorChannel <- Error{Type: NoError}
		} else if err != nil {
			errorChannel <- Error{Type: GrpcError, Message: err.Error()}
		}
	}()

	result := <-resultChannel
	error := <-errorChannel

	return NewResult(&result, &error)
}

func getStackTrace(log *l.Log) {
	fakeError := errors.Error{}
	stack := fakeError.StackFrames()

	for _, frame := range stack {
		codeResult := getCode(frame.File, uint32(frame.LineNumber))
		code := ""

		if codeResult.IsOk() {
			code = *codeResult.Ok()
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
}

func getCode(filePath string, lineNumber uint32) Result[string] {
	file, err := os.Open(filePath)

	if err != nil {
		return NewErrResult[string](IoError, err.Error())
	}

	defer file.Close()

	contentBytes, err := io.ReadAll(file)

	if err != nil {
		return NewErrResult[string](IoError, err.Error())
	}

	content := string(contentBytes)
	lines := strings.Split(content, "\n")

	if len(lines) < int(lineNumber) {
		return NewErrResult[string](LineNumTooLargeError, "Line number is too large for this file.")
	} else if int(lineNumber) <= 0 {
		return NewErrResult[string](LineNumZeroError, "Line number is zero or negative.")
	}

	line := lines[lineNumber-1]

	return NewOkResult(&line)
}

// TODO: Add code snippet retrieval function
