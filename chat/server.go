package chat

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
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
                fmt.Fprintln(newClient.conn, "Username already taken, please reconnect with a different username.")
                newClient.conn.Close()
            } else {
                usernameSet[newClient.username] = true
                clients = append(clients, newClient)
                // Announce the new user has joined
                broadcastMessage(fmt.Sprintf("System: %s has joined the chat.", newClient.username))
            }
            clientMux.Unlock()
        case exClient := <-removing:
            clientMux.Lock()
            for i, c := range clients {
                if c.conn == exClient.conn {
                    clients = append(clients[:i], clients[i+1:]...)
                    delete(usernameSet, c.username)
                    // Announce that the user has left the chat
                    broadcastMessage(fmt.Sprintf("System: %s has left the chat.", c.username))
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
    // Expect the first message to be the username
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
    listener, err := net.Listen("tcp", "0.0.0.0:9999")
    if err != nil {
        fmt.Println("Failed to start server:", err)
        return
    }
    defer listener.Close()
    fmt.Println("Server started on all interfaces, port 9999")

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