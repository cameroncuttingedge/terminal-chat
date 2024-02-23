package chat

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/cameroncuttingedge/terminal-chat/util"
)

type client struct {
    conn     net.Conn
    username string
}

var (
    clients   []client
    adding    = make(chan client)
    removing  = make(chan client)
    messages  = make(chan string)
    clientMux sync.Mutex
    usernameSet = make(map[string]bool) // Track usernames to ensure uniqueness
    colors    = []string{"31", "32", "33", "34", "35", "36", "37"} // ANSI color codes
)


func broadcast() {
    for {
        select {
        case msg := <-messages:
            clientMux.Lock()
            for _, c := range clients {
                fmt.Fprintln(c.conn, msg)
            }
            clientMux.Unlock()
        case newClient := <-adding:
            clientMux.Lock()
            // Ensure username is unique
            if _, exists := usernameSet[newClient.username]; exists {
                // If username is taken, send a message to the new client and remove them
                fmt.Fprintln(newClient.conn, "SYSTEM_MESSAGE:UsernameTaken")
                newClient.conn.Close()
            } else {
                usernameSet[newClient.username] = true
                clients = append(clients, newClient)
                // Announce the new user has joined
                broadcastMessage(fmt.Sprintf("Robot: %s has joined the chat.", newClient.username))
            }
            clientMux.Unlock()
        case exClient := <-removing:
            clientMux.Lock()
            for i, c := range clients {
                if c.conn == exClient.conn {
                    clients = append(clients[:i], clients[i+1:]...)
                    delete(usernameSet, c.username)
                    // Announce that the user has left the chat
                    broadcastMessage(fmt.Sprintf("Robot: %s has left the chat.", c.username))
                    break
                }
            }
            clientMux.Unlock()
        }
    }
}

func broadcastMessage(message string) {
    // Helper function to send a message to all clients
    for _, c := range clients {
        fmt.Fprintln(c.conn, message)
    }
}

func handleConnection(conn net.Conn) {
    // Temporary client object; username will be set upon receiving the first message
    newClient := client{conn: conn}

    reader := bufio.NewReader(conn)
    username, err := reader.ReadString('\n')
    if err != nil {
        conn.Close()
        return
    }
    username = strings.TrimSpace(username)
    newClient.username = username

    adding <- newClient // Add the client with the username

    defer func() {
        removing <- newClient
    }()

    for {
        message, err := reader.ReadString('\n')
        if err != nil {
            break // Connection closed or error occurred
        }
        messages <- strings.TrimSpace(message)
    }
}

func StartServer() {
    port := flag.Int("port", 9999, "The port number on which the server listens")
    flag.Parse()

    portStr := fmt.Sprintf(":%d", *port)

    listener, err := net.Listen("tcp", "0.0.0.0"+portStr)
    if err != nil {
        fmt.Println("Failed to start server:", err)
        return
    }
    
    defer listener.Close()
    
    localIP := util.GetLocalIP()
    fmt.Printf("Server started on %s%s\n", localIP, portStr)
    fmt.Printf("Clients can connect using: nc %s %d\n", localIP, *port)
    fmt.Printf("Use these flags -ip=%s -port=%d\n", localIP, *port)


    go broadcast()

    for {
        conn, err := listener.Accept()
        if err != nil {
            fmt.Println("Error accepting connection:", err)
            continue
        }
        go handleConnection(conn)
    }
}