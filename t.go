package chat

import (
	"bufio"
	"fmt"
	"hash/fnv"
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

// Simple hash function to assign colors based on username
func hashUsernameToColor(username string) string {
    h := fnv.New32a()
    h.Write([]byte(username))
    return colors[h.Sum32()%uint32(len(colors))]
}

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
    reader := bufio.NewReader(conn)
    username, err := reader.ReadString('\n')
    if err != nil {
        conn.Close()
        return
    }
    username = strings.TrimSpace(username)

    // Check for unique username
    clientMux.Lock()
    if _, exists := usernameSet[username]; exists {
        fmt.Fprintln(conn, "Username already taken, please reconnect with a different username.")
        conn.Close()
        clientMux.Unlock()
        return
    }
    usernameSet[username] = true
    clientMux.Unlock()

    // Assign a color
    color := hashUsernameToColor(username)
    newClient := client{conn: conn, username: username, color: color}

    // Announce the new user has joined
    adding <- newClient
    broadcast(fmt.Sprintf("\x1b[%smSystem: %s has joined the chat.\x1b[0m", color, username))

    defer func() {
        removing <- newClient
        broadcast(fmt.Sprintf("\x1b[%smSystem: %s has left the chat.\x1b[0m", color, username))
    }()

    for {
        message, err := reader.ReadString('\n')
        if err != nil {
            break
        }
        // Broadcast message with colored username
        broadcast(fmt.Sprintf("\x1b[%sm%s\x1b[0m: %s", color, username, strings.TrimSpace(message)))
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