package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	// Definizione delle flag CLI
	listenPort := flag.String("l", "", "Porta su cui ascoltare (es: 8080)")
	connectAddr := flag.String("c", "", "Indirizzo IP:porta a cui connettersi")

	flag.Parse()

	if *listenPort != "" {
		fmt.Printf("Listening on 0.0.0.0:%s\n", *listenPort)
		// TODO: Avviare server TCP
	} else if *connectAddr != "" {
		fmt.Printf("Connected to %s\n", *connectAddr)
		// TODO: Avviare client TCP
	} else {
		fmt.Println("Uso: ./oneone -l <porta> oppure ./oneone -c <IP:porta>")
		os.Exit(1)
	}
}