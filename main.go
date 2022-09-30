package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/haplesspanda/haplessbot/commands"
	"github.com/haplesspanda/haplessbot/gateway"
)

var logfile *os.File

func init() {
	logfilename := fmt.Sprintf("log/log-%d.txt", time.Now().UnixMilli())
	err := os.MkdirAll("log", os.ModePerm)
	if err != nil {
		panic(err)
	}
	logfile, err = os.Create(logfilename)
	if err != nil {
		panic(err)
	}
	log.SetOutput(logfile)
}

func main() {
	fmt.Println("Starting up bot operations...")

	defineCommands := flag.String("define_commands", "", "Comma-separated list of commands to push, if any")
	flag.Parse()

	parsedCommands := strings.Split(*defineCommands, ",")

	if len(parsedCommands) != 0 && !(len(parsedCommands) == 1 && parsedCommands[0] == "") {
		commands.DefineCommands(parsedCommands)
	} else {
		log.Println("No commands to push, skipping")
	}

	gateway.StartConnection()

	// Cleanup logic below.
	err := logfile.Close()
	if err != nil {
		panic(err)
	}

	fmt.Println("Done")
}
