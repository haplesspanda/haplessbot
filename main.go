package main

import (
	"flag"
	"log"
	"strings"

	"github.com/haplesspanda/haplessbot/commands"
	"github.com/haplesspanda/haplessbot/gateway"
)

func main() {
	log.Println("Starting up bot operations...")

	defineCommands := flag.String("define_commands", "", "Comma-separated list of commands to push, if any")
	flag.Parse()

	parsedCommands := strings.Split(*defineCommands, ",")

	if len(parsedCommands) != 0 && !(len(parsedCommands) == 1 && parsedCommands[0] == "") {
		commands.DefineCommands(parsedCommands)
	} else {
		log.Println("No commands to push, skipping")
	}

	gateway.StartConnection()

	log.Println("Done")
}
