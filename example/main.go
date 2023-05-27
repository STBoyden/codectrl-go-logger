package main

import (
	"github.com/Authentura/codectrl-go-logger"
)

func finalLayer() {
	logger := codectrl.Logger{}

	logger.Log("Hello, world!")
}

func layer3() {
	finalLayer()
}

func layer2() {
	layer3()
}

func layer1() {
	layer2()
}

func main() {
	layer1()
}
