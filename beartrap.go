/*
 * Copyright (c) 2015, Chris Benedict <chrisbdaemon@gmail.com>
 * All rights reserved.
 *
 * Licensing terms are located in LICENSE file.
 */

package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"beartrap/alert"
	"beartrap/broadcast"
	"beartrap/config"
	"beartrap/handler"
	"beartrap/trap"
	getopt "github.com/kesselborn/go-getopt"
)

func main() {
	options := getOptions()

	cfg, err := config.New(options["config"].String)
	if err != nil {
		log.Fatal(err)
	}

	trapParams, err := cfg.TrapParams()
	if err != nil {
		log.Fatalf("Error reading traps: %s", err)
	}

	var broadcast broadcast.Broadcast

	handlerParams, err := cfg.HandlerParams()
	if err != nil {
		log.Fatalf("Error reading handlers: %s", err)
	}

	handlers, err := initHandlers(handlerParams, &broadcast)
	if err != nil {
		log.Fatalf("Error initializing handlers: %s", err)
	}
	errors := validateHandlers(handlers)
	if len(errors) > 0 {
		displayErrors(errors)
		os.Exit(-1)
	} else {
		startHandlers(handlers)
	}

	// Create and validate traps
	traps, err := initTraps(trapParams, &broadcast)
	if err != nil {
		log.Fatalf("Error initializing traps: %s", err)
	}
	errors = validateTraps(traps)

	// If validation failed, report and quit,
	// if not, turn traps on
	if len(errors) > 0 {
		displayErrors(errors)
		os.Exit(-1)
	} else {
		startTraps(traps)

		// Hack to let traps run till I create a better mainloop
		for {
			time.Sleep(500 * time.Second)
		}
	}
}

// displayErrors takes a slice of errors and prints them to the screen
func displayErrors(errors []error) {
	for i := range errors {
		log.Println(errors[i])
	}
}

// initTraps take in a list of trap parameters, creates trap objects
// that are returned
func initTraps(trapParams []config.Params, b *broadcast.Broadcast) ([]trap.Interface, error) {
	traps := []trap.Interface{}

	for i := range trapParams {
		trap, err := trap.New(trapParams[i], b)
		if err != nil {
			return nil, err
		}

		traps = append(traps, trap)
	}

	return traps, nil
}

func validateTraps(traps []trap.Interface) []error {
	var errors []error
	for i := range traps {
		errors = append(errors, traps[i].Validate()...)
	}
	return errors
}

// startTraps takes a slice of traps and starts them in a goroutine
// TODO: Allow them to be stopped
func startTraps(traps []trap.Interface) {
	for i := range traps {
		go traps[i].Start()
	}
}

// initHandlers take in a list of handler parameters, creates handler objects
// that are returned
func initHandlers(handlerParams []config.Params, b *broadcast.Broadcast) ([]handler.Interface, error) {
	handlers := []handler.Interface{}

	for i := range handlerParams {
		c := make(chan alert.Alert)
		handler, err := handler.New(handlerParams[i], c)
		if err != nil {
			return nil, err
		}

		handler.Init()

		b.AddReceiver(c)
		handlers = append(handlers, handler)
	}

	return handlers, nil
}

func validateHandlers(handlers []handler.Interface) []error {
	var errors []error
	for i := range handlers {
		errors = append(errors, handlers[i].Validate()...)
	}
	return errors
}

// startHandlers takes a slice of handlers and starts them in a goroutine
// TODO: Allow them to be stopped
func startHandlers(handlers []handler.Interface) {
	for i := range handlers {
		go handlers[i].Start()
	}
}

// Parse commandline arguments into getopt object
func getOptions() map[string]getopt.OptionValue {
	optionDefinition := getopt.Options{
		Description: "Beartrap v0.3 by Chris Benedict <chrisbdaemon@gmail.com>",
		Definitions: getopt.Definitions{
			{"config|c|BEARTRAP_CONFIG", "configuration file", getopt.Required, ""},
		},
	}

	options, _, _, err := optionDefinition.ParseCommandLine()

	help, wantsHelp := options["help"]

	if err != nil || wantsHelp {
		exitCode := 0

		switch {
		case wantsHelp && help.String == "usage":
			fmt.Print(optionDefinition.Usage())
		case wantsHelp && help.String == "help":
			fmt.Print(optionDefinition.Help())
		default:
			fmt.Println("**** Error: ", err.Error(), "\n", optionDefinition.Help())
			exitCode = err.ErrorCode
		}
		os.Exit(exitCode)
	}

	return options
}
