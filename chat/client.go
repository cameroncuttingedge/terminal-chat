package chat

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/cameroncuttingedge/terminal-chat/alert"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ChatUI encapsulates the UI elements of the chat application.
type ChatUI struct {
    App        *tview.Application
    ChatView   *tview.TextView
    InputField *tview.InputField
}

func StartClient() {
    log.Println("Starting client application...")
    serverIP := flag.String("ip", "127.0.0.1", "The IP address of the server to connect to.")
	serverPort := flag.String("port", "9999", "The port of the server to connect to.")

	flag.Parse()

    app := tview.NewApplication()

    // Show login screen and get credentials
    username := showFormScreen(app, "Enter your username", "Username")
    password := showFormScreen(app, "Enter your password", "Password")

    // Initialize the UI components
    chatUI := setupUIComponents(app, username, password)

    // Connect to server and handle chat session
    startChatSession(chatUI, username, *serverIP, *serverPort)

    
}

func setupUIComponents(app *tview.Application, username, password string) *ChatUI {
    chatUI := &ChatUI{
        App: app,
    }

    chatUI.ChatView = tview.NewTextView()

    // Initialize the chat view
    chatUI.ChatView = tview.NewTextView()
    chatUI.ChatView.SetDynamicColors(true)
    chatUI.ChatView.SetRegions(true)
    chatUI.ChatView.SetScrollable(true)
    chatUI.ChatView.SetBackgroundColor(tcell.ColorDefault)
    chatUI.ChatView.SetBorder(false)
    chatUI.ChatView.SetTitle(" Chat ")

    // Initialize the input field
    chatUI.InputField = tview.NewInputField()
    chatUI.InputField.SetLabel(fmt.Sprintf("%s: ", username))
    chatUI.InputField.SetFieldWidth(0)
    chatUI.InputField.SetFieldBackgroundColor(tcell.ColorDefault)
    chatUI.InputField.SetBorder(false)
    chatUI.InputField.SetTitle(" Input ")

    // No need for SetInputCapture if we are using default scrolling behavior

    // Setup the UI layout
    flex := tview.NewFlex().
        SetDirection(tview.FlexRow).
        AddItem(chatUI.ChatView, 0, 1, false). 
        AddItem(chatUI.InputField, 1, 0, true)

    app.SetRoot(flex, true).SetFocus(chatUI.InputField)

    return chatUI
}

func connectToServer(serverIp, serverPort string) (net.Conn, error) {
    log.Printf("Attempting to connect to server at %s:%s", serverIp, serverPort)
    conn, err := net.Dial("tcp", serverIp+":"+serverPort)
    if err != nil {
        log.Printf("Failed to connect to server: %v", err)
        return nil, err
    }
    log.Println("Successfully connected to server")
    return conn, nil
}


func sendUsername(conn net.Conn, username string) error {
    log.Printf("Sending username: %s", username)
    _, err := fmt.Fprintf(conn, "%s\n", username)
    if err != nil {
        log.Printf("Failed to send username: %v", err)
    } else {
        log.Println("Username sent successfully")
    }
    return err
}


func setupMessageSending(ui *ChatUI, conn net.Conn, username string) {
    ui.InputField.SetDoneFunc(func(key tcell.Key) {
        if key == tcell.KeyEnter {
            message := ui.InputField.GetText()
            if message != "" {
                log.Printf("Attempting to send message: %s", message)
                _, err := fmt.Fprintf(conn, "%s: %s\n", username, message)
                if err != nil {
                    log.Printf("Error sending message: %v", err)
                } else {
                    log.Println("Message sent successfully")
                }
                ui.InputField.SetText("")
                alert.PlaySoundAsync("out.wav")
            }
        }
    })
}


func handleIncomingMessages(conn net.Conn, ui *ChatUI, username string) {
    scanner := bufio.NewScanner(conn)
    for scanner.Scan() {
        text := scanner.Text()

        log.Printf("Received text from server: %s", text)

        // Check if the username is already taken
        if strings.HasPrefix(text, "SYSTEM_MESSAGE:UsernameTaken") {
            fmt.Fprintln(tview.ANSIWriter(ui.ChatView), "[red]Username already taken. Please restart the client and choose a different username.[-]")
            time.Sleep(2 * time.Second)
            conn.Close()
            ui.App.Stop()
            return
        }
        ui.App.QueueUpdateDraw(func() {
            parts := strings.SplitN(text, ": ", 2)
            if len(parts) == 2 && !isUsernameContained(parts[0], username) {
                alert.PlaySoundAsync("in.wav")
            }
            fmt.Fprintln(tview.ANSIWriter(ui.ChatView), text)
        })
    }
}


func startChatSession(ui *ChatUI, username string, serverIp string, serverPort string) {
    
    // Connect to the mothership
    conn, err := connectToServer(serverIp, serverPort)
    if err != nil {
        fmt.Fprintf(tview.ANSIWriter(ui.ChatView), "[red]Failed to connect to server: %v\n", err)
        return
    }
    defer conn.Close()

    // Sending username to server
    if err := sendUsername(conn, username); err != nil {
        fmt.Fprintf(tview.ANSIWriter(ui.ChatView), "[red]Failed to send username: %v\n", err)
        return
    }

    // Setting up message sending functionality
    setupMessageSending(ui, conn, username)


    // Handling incoming messages
    go handleIncomingMessages(conn, ui, username)


    // Running the tview application
    if err := ui.App.Run(); err != nil {
        fmt.Fprintf(os.Stderr, "Error running application: %v\n", err)
        os.Exit(1)
    }
}


func showFormScreen(app *tview.Application, title, label string) string {
    var input string
    form := tview.NewForm().
        AddInputField(label, "", 20, nil, func(text string) {
            input = text
        }).
        AddButton("Submit", func() {
            app.Stop()
        }).
        SetCancelFunc(func() {
            app.Stop()
        })
    form.SetBorder(true).SetTitle(title).SetTitleAlign(tview.AlignLeft).SetBackgroundColor(tcell.ColorDefault)

    if err := app.SetRoot(form, true).SetFocus(form).Run(); err != nil {
        fmt.Fprintf(os.Stderr, "Error running application: %v\n", err)
        os.Exit(1)
    }

    return input
}


func isUsernameContained(encodedStr, username string) bool {
	// Regex to find and remove color encoding like [green]...[-]
	re := regexp.MustCompile(`\[[^\[\]]*\]`)
	cleanStr := re.ReplaceAllString(encodedStr, "")

	// Now check if the username is contained within the cleaned string
	return strings.Contains(cleanStr, username)
}