package codectrl

import (
	"context"
	"fmt"
	"reflect"

	b "github.com/Authentura/codectrl-go-protobufs/data/backtrace_data"
	l "github.com/Authentura/codectrl-go-protobufs/data/log"
	logs_service "github.com/Authentura/codectrl-go-protobufs/logs_service"
	grpc "google.golang.org/grpc"
)

type create_log_params struct {
	surround                 uint32
	function_name            string
	function_name_occurences hashbag[string]
}

func create_log(message string, params ...create_log_params) l.Log {
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
	Log(message string, params ...LoggerParams) Result[chan logs_service.RequestResult]
	log(log l.Log, host string, port string) Result[chan logs_service.RequestResult]
}

func (logger Logger) Log(message string, params ...LoggerParams) Result[chan logs_service.RequestResult] {
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

	log := create_log(message, create_log_params{surround: surround})

	return logger.log(log, host, port)
}

func (logger Logger) log(log l.Log, host string, port string) Result[chan logs_service.RequestResult] {
	connection, err := grpc.Dial(fmt.Sprintf("%s:%s", host, port), grpc.WithInsecure())

	if err != nil {
		return NewErrResult[chan logs_service.RequestResult](GrpcError, err.Error())
	}

	client := logs_service.NewLogClientClient(connection)

	r := make(chan logs_service.RequestResult)
	error_chan := make(chan Error)

	go func() {
		defer connection.Close()

		result, err := client.SendLog(context.Background(), &log)

		if result != nil {
			r <- *result
			error_chan <- Error{Type: NoError}
		} else if err != nil {
			error_chan <- Error{Type: GrpcError, Message: err.Error()}
		}
	}()

	error := <-error_chan

	return NewResult(&r, &error)
}
