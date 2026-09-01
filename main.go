package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"

	"golang.org/x/term"
)

// Gestore del terminale con supporto al ridisegno concorrente
type TerminalManager struct {
	mu       sync.Mutex
	inputBuf []rune
	oldState *term.State
	stdinFd  int
}

func NewTerminalManager() (*TerminalManager, error) {
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return nil, err
	}

	return &TerminalManager{
		inputBuf: make([]rune, 0),
		oldState: oldState,
		stdinFd:  fd,
	}, nil
}

func (tm *TerminalManager) Restore() {
	if tm.oldState != nil {
		_ = term.Restore(tm.stdinFd, tm.oldState)
	}
}

// Cancella la riga corrente, stampa il messaggio remoto e ripristina l'input locale
func (tm *TerminalManager) PrintIncoming(msg string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Sequenza ANSI: \r torna a inizio riga, \033[K cancella da lì a fine riga
	fmt.Print("\r\033[K")
	fmt.Printf("<0 %s\r\n", msg)
	fmt.Printf("0> %s", string(tm.inputBuf))
}

func main() {
	listenPort := flag.String("l", "", "Porta su cui ascoltare")
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

func startServer(port string) {
	listener, err := net.Listen("tcp", "0.0.0.0:"+port)
	if err != nil {
		fmt.Printf("Errore avvio server: %v\n", err)
		os.Exit(1)
	}
	defer listener.Close()

	fmt.Printf("Listening on 0.0.0.0:%s\r\n0> ", port)

	conn, err := listener.Accept()
	if err != nil {
		fmt.Printf("Errore connessione: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Printf("\r\033[KAccepted connection from %s\r\n", conn.RemoteAddr().String())
	handleChat(conn)
}

func startClient(addr string) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Printf("Errore connessione a %s: %v\n", addr, err)
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Printf("Connected to %s\r\n", addr)
	handleChat(conn)
}

func handleChat(conn net.Conn) {
	tm, err := NewTerminalManager()
	if err != nil {
		fmt.Printf("Impossibile impostare RAW mode: %v\n", err)
		return
	}
	defer tm.Restore()

	done := make(chan struct{})

	// Goroutine per la lettura dalla rete
	go func() {
		reader := bufio.NewReader(conn)
		for {
			msg, err := reader.ReadString('\n')
			if err != nil {
				close(done)
				return
			}
			msg = strings.TrimRight(msg, "\r\n")
			if msg != "" {
				tm.PrintIncoming(msg)
			}
		}
	}()

	// Goroutine per la lettura dell'input utente (carattere per carattere)
	go func() {
		fmt.Print("0> ")
		buf := make([]byte, 3)

		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				close(done)
				return
			}

			if n == 1 {
				b := buf[0]

				switch b {
				case 3: // Ctrl+C
					close(done)
					return

				case 13, 10: // Enter (\r o \n)
					tm.mu.Lock()
					line := string(tm.inputBuf)
					tm.inputBuf = tm.inputBuf[:0]
					fmt.Print("\r\n0> ")
					tm.mu.Unlock()

					if strings.TrimSpace(line) != "" {
						_, err := conn.Write([]byte(line + "\n"))
						if err != nil {
							close(done)
							return
						}
					}

				case 127, 8: // Backspace
					tm.mu.Lock()
					if len(tm.inputBuf) > 0 {
						tm.inputBuf = tm.inputBuf[:len(tm.inputBuf)-1]
						fmt.Print("\r\033[K0> " + string(tm.inputBuf))
					}
					tm.mu.Unlock()

				default: // Caratteri stampabili
					if b >= 32 {
						tm.mu.Lock()
						tm.inputBuf = append(tm.inputBuf, rune(b))
						fmt.Print(string(b))
						tm.mu.Unlock()
					}
				}
			}
		}
	}()

	<-done
}