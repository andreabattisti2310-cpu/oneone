package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
)

func main() {
	// Definizione delle flag
	listenPort := flag.String("l", "", "Porta su cui ascoltare (es: 8080)")
	connectAddr := flag.String("c", "", "Indirizzo IP:porta a cui connettersi")

	flag.Parse()

	if *listenPort != "" {
		startServer(*listenPort)
	} else if *connectAddr != "" {
		startClient(*connectAddr)
	} else {
		fmt.Println("Uso: ./oneone -l <porta> oppure ./oneone -c <IP:porta>")
		os.Exit(1)
	}
}

// Modalità Server (-l <porta>)
func startServer(port string) {
	listener, err := net.Listen("tcp", "0.0.0.0:"+port)
	if err != nil {
		fmt.Printf("Errore durante l'avvio del server: %v\n", err)
		os.Exit(1)
	}
	defer listener.Close()

	fmt.Printf("Listening on 0.0.0.0:%s\n", port)

	conn, err := listener.Accept()
	if err != nil {
		fmt.Printf("Errore nell'accettare la connessione: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Printf("\rAccepted connection from %s\n", conn.RemoteAddr().String())

	handleChat(conn)
}

// Modalità Client (-c <IP:porta>)
func startClient(addr string) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Printf("Errore di connessione a %s: %v\n", addr, err)
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Printf("Connected to %s\n", addr)

	handleChat(conn)
}

// Gestione della comunicazione bidirezionale
func handleChat(conn net.Conn) {
	done := make(chan struct{})

	// Goroutine per leggere i messaggi in arrivo dal socket TCP
	go func() {
		reader := bufio.NewReader(conn)
		for {
			message, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("\nConnessione chiusa dall'altro lato.")
				close(done)
				return
			}
			fmt.Printf("<0 %s", message)
			fmt.Print("0> ")
		}
	}()

	// Goroutine per leggere l'input da tastiera e inviarlo
	go func() {
		stdinReader := bufio.NewReader(os.Stdin)
		for {
			fmt.Print("0> ")
			input, err := stdinReader.ReadString('\n')
			if err != nil {
				close(done)
				return
			}

			_, err = conn.Write([]byte(input))
			if err != nil {
				fmt.Println("Errore nell'invio del messaggio.")
				close(done)
				return
			}
		}
	}()

	<-done
}