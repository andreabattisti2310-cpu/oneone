package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/rivo/uniseg"
	"golang.org/x/term"
)

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

// CalculateRows determina quante righe occupano la stringa nel terminale in base alla larghezza dello schermo.
func CalculateRows(promptAndText string, termWidth int) int {
	if termWidth <= 0 {
		termWidth = 80
	}
	// Calcola le colonne visive effettive (non i byte)
	displayWidth := uniseg.StringWidth(promptAndText)
	if displayWidth == 0 {
		return 1
	}
	return (displayWidth + termWidth - 1) / termWidth
}

// Cancella l'input a schermo basandosi sui caratteri visivi reali
func (tm *TerminalManager) clearInputLineLocked() {
	width, _, err := term.GetSize(tm.stdinFd)
	if err != nil || width <= 0 {
		width = 80
	}

	fullLine := "0> " + string(tm.inputBuf)
	rows := CalculateRows(fullLine, width)

	// Riposiziona il cursore all'inizio dell'input e cancella verso il basso
	for i := 1; i < rows; i++ {
		fmt.Print("\033[F") // Sale di una riga
	}
	fmt.Print("\r\033[J") // Cancella da inizio riga a fondo schermo
}

func (tm *TerminalManager) PrintIncoming(msg string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.clearInputLineLocked()
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
	var closeOnce sync.Once
	safeClose := func() {
		closeOnce.Do(func() {
			close(done)
		})
	}

	go func() {
		defer safeClose()
		reader := bufio.NewReader(conn)
		for {
			msg, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			msg = strings.TrimRight(msg, "\r\n")
			if msg != "" {
				tm.PrintIncoming(msg)
			}
		}
	}()

	go func() {
		defer safeClose()
		fmt.Print("0> ")

		buf := make([]byte, 1024)
		var pending []byte

		for {
			n, err := os.Stdin.Read(buf)
			if err != nil && err != io.EOF {
				return
			}
			if n == 0 {
				return
			}

			pending = append(pending, buf[:n]...)

			for len(pending) > 0 {
				r, size := utf8.DecodeRune(pending)
				if r == utf8.RuneError && size == 1 && !utf8.FullRune(pending) {
					break
				}
				pending = pending[size:]

				switch r {
				case 3: // Ctrl+C
					return

				case '\r', '\n':
					tm.mu.Lock()
					line := string(tm.inputBuf)
					tm.inputBuf = tm.inputBuf[:0]
					fmt.Print("\r\n0> ")
					tm.mu.Unlock()

					if strings.TrimSpace(line) != "" {
						_, writeErr := conn.Write([]byte(line + "\n"))
						if writeErr != nil {
							return
						}
					}

				case 127, 8: // Backspace
					tm.mu.Lock()
					if len(tm.inputBuf) > 0 {
						tm.inputBuf = tm.inputBuf[:len(tm.inputBuf)-1]
						tm.clearInputLineLocked()
						fmt.Printf("0> %s", string(tm.inputBuf))
					}
					tm.mu.Unlock()

				default:
					if r >= 32 {
						tm.mu.Lock()
						tm.inputBuf = append(tm.inputBuf, r)
						fmt.Print(string(r))
						tm.mu.Unlock()
					}
				}
			}
		}
	}()

	<-done
}
