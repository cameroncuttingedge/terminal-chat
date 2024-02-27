package chat

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"

	"github.com/cameroncuttingedge/terminal-chat/util"
)

type client struct {
    conn     net.Conn
    username string
    color    string
}

var (
    clients   []client
    adding    = make(chan client)
    removing  = make(chan client)
    messages  = make(chan string)
    clientMux sync.Mutex
    usernameSet = make(map[string]bool) // Track usernames to ensure uniqueness
    colors = []string{
        "[green]",     
        "[yellow]",    
        "[blue]",      
        "[fuchsia]",   
        "[cyan]",     
        "[white]",   
        "[lightred]", 
        "[lightgreen]",
        "[lightyellow]",
        "[lightblue]", 
        "[lightmagenta]",
        "[lightcyan]",
        "[lightgray]", 
    }
    
)

func broadcast() {
    for {
        select {
        case msg := <-messages:
            log.Printf("[Server] Received message to broadcast: %s", msg)
            broadcastMessage(msg, "")
        case newClient := <-adding:
            prepareClientAddition(newClient)
        case exClient := <-removing:
            prepareClientRemoval(exClient)
        }
    }
}

func prepareClientAddition(newClient client) {
    clientMux.Lock()
    if _, exists := usernameSet[newClient.username]; exists {
        clientMux.Unlock() // Unlock before network I/O
        log.Printf("[Server] Username %s is taken, sending UsernameTaken message", newClient.username)
        fmt.Fprintln(newClient.conn, "SYSTEM_MESSAGE:UsernameTaken")
        newClient.conn.Close()
    } else {
        usernameSet[newClient.username] = true
        clients = append(clients, newClient)
        clientMux.Unlock() // Unlock before broadcasting to avoid holding the lock during I/O
        broadcastMessage(fmt.Sprintf("Robot: %s has joined the chat.", newClient.username), "SYSTEM")
    }
}

func prepareClientRemoval(exClient client) {
    clientMux.Lock()
    for i, c := range clients {
        if c.conn == exClient.conn {
            clients = append(clients[:i], clients[i+1:]...)
            delete(usernameSet, exClient.username)
            break
        }
    }
    clientMux.Unlock()
    broadcastMessage(fmt.Sprintf("Robot: %s has left the chat.", exClient.username), "SYSTEM")
}


func broadcastMessage(message string, messageType string) {
    clientMux.Lock()
    defer clientMux.Unlock()

    formattedMessage := formatMessage(message, messageType)

    for _, c := range clients {
        _, err := fmt.Fprintln(c.conn, formattedMessage)
        if err != nil {
            log.Printf("Error broadcasting to client %s: %v", c.username, err)
        } else {
            log.Printf("[Server] Broadcasting message: %s to client: %s", formattedMessage, c.username)
        }
    }
}



func formatMessage(message, messageType string) string {
    if messageType == "SYSTEM" {
        return fmt.Sprintf("[red]%s[-]", message)
    } else { 
        parts := strings.SplitN(message, ": ", 2)
        if len(parts) == 2 {
            var userColor string
            // Find the user's color
            for _, c := range clients {
                if c.username == parts[0] {
                    userColor = c.color
                    break
                }
            }
            if userColor == "" { // Default color if not found
                userColor = "[white]"
            }
            return fmt.Sprintf("%s%s[-]: %s", userColor, parts[0], parts[1])
        }
        // Fallback for unexpected message format
        return message
    }
}



func handleConnection(conn net.Conn) {
    // Temporary client object; username will be set upon receiving the first message
    newClient := client{conn: conn}

    // Assign a color to the new client based on the current number of clients
    newClient.color = colors[len(clients)%len(colors)]

    reader := bufio.NewReader(conn)
    username, err := reader.ReadString('\n')
    if err != nil {
        log.Printf("Error during username read: %v", err)
        conn.Close()
        return
    }

    username = strings.TrimSpace(username)
    newClient.username = username
    log.Printf("[Server] New client '%s' connected", newClient.username)
    

    adding <- newClient

    log.Printf("New client connected: %s", newClient.username)

    defer func() {
        removing <- newClient
    }()

    for {
        message, err := reader.ReadString('\n')
        log.Printf("[Server] Received raw message from '%s': %s", newClient.username, message)
        if err != nil {
            log.Printf("Error reading from client %s: %v", newClient.username, err)
            break // Connection closed or error occurred
        }
        trimmedMessage := strings.TrimSpace(message)

        messageContent := getMessageContentAfterColon(trimmedMessage)

        if strings.HasPrefix(messageContent, "!man") {
            handleSpecialCommand(newClient, "man")
            continue
        } else if strings.HasPrefix(messageContent, "!party") {
            handleSpecialCommand(newClient, "party")
            continue
        }
        log.Printf("[Server] Sending message from '%s' to channel: %s", newClient.username, trimmedMessage)
        messages <- trimmedMessage
        log.Printf("[Server] Message sent to channel from '%s'", newClient.username)
    }

    log.Printf("Client disconnected: %s", newClient.username)
}

func getMessageContentAfterColon(message string) string {
    if idx := strings.Index(message, ":"); idx != -1 {
        return strings.TrimSpace(message[idx+1:])
    }
    return message 
}


func handleSpecialCommand(c client, action string) {
    var specialMessage string

    switch action {
    case "man":
        specialMessage = "Robot: To scroll through the chat, use the arrow keys or your mouse wheel. To focus on the input section, press Tab."
    case "party":
        specialMessage = `
░░░░░░░░▄▄▄▀▀▀▄▄███▄░░░░░░░░░░░░░░
░░░░░▄▀▀░░░░░░░▐░▀██▌░░░░░░░░░░░░░
░░░▄▀░░░░▄▄███░▌▀▀░▀█░░░░░░░░░░░░░
░░▄█░░▄▀▀▒▒▒▒▒▄▐░░░░█▌░░░░░░░░░░░░
░▐█▀▄▀▄▄▄▄▀▀▀▀▌░░░░░▐█▄░░░░░░░░░░░
░▌▄▄▀▀░░░░░░░░▌░░░░▄███████▄░░░░░░
░░░░░░░░░░░░░▐░░░░▐███████████▄░░░
░░░░░le░░░░░░░▐░░░░▐█████████████▄
░░░░toucan░░░░░░▀▄░░░▐█████████████▄
░░░░░░has░░░░░░░░▀▄▄███████████████
░░░░░arrived░░░░░░░░░░░░█▀██████░░`
    default:
        specialMessage = "Unknown command."
    }

    specialMessage = formatMessage(specialMessage, "SYSTEM")

    fmt.Fprintln(c.conn, specialMessage)
}


func StartServer() {
    port := flag.Int("port", 9999, "The port number on which the server listens")
    flag.Parse()
    //go monitorChannelState()
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

