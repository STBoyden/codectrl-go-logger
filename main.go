package codectrl

import (
	"context"
	"fmt"
	"reflect"

	b "github.com/Authentura/codectrl-go-protobufs/data/backtrace_data"
	l "github.com/Authentura/codectrl-go-protobufs/data/log"
	logsService "github.com/Authentura/codectrl-go-protobufs/logs_service"
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

	return log
}

type LoggerParams struct {
	surround uint32
	host     string
	port     string
}

type Logger struct{}

type loggerInterface interface {
	Log(message string, params ...LoggerParams) Result[chan logsService.RequestResult]
	log(log l.Log, host string, port string) Result[chan logsService.RequestResult]
}

func (logger Logger) Log(message string, params ...LoggerParams) Result[chan logsService.RequestResult] {
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

func (logger Logger) log(log l.Log, host string, port string) Result[chan logsService.RequestResult] {
	connection, err := grpc.Dial(fmt.Sprintf("%s:%s", host, port), grpc.WithInsecure())

	if err != nil {
		return NewErrResult[chan logsService.RequestResult](GrpcError, err.Error())
	}

	client := logsService.NewLogClientClient(connection)

	r := make(chan logsService.RequestResult)
	errorChannel := make(chan Error)

	go func() {
		defer connection.Close()

		result, err := client.SendLog(context.Background(), &log)

		if result != nil {
			r <- *result
			errorChannel <- Error{Type: NoError}
		} else if err != nil {
			errorChannel <- Error{Type: GrpcError, Message: err.Error()}
		}
	}()

	error := <-errorChannel

	return NewResult(&r, &error)
}
