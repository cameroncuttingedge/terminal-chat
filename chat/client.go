package chat

import (
	"bufio"
	"flag"
	"fmt"
	"hash/fnv"
	"net"
	"os"
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
    chatUI.ChatView.SetDynamicColors(false)
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



func startChatSession(ui *ChatUI, username string, serverIp string, serverPort string) {
    conn, err := net.Dial("tcp", serverIp + ":" + serverPort)
    if err != nil {
        fmt.Fprintf(tview.ANSIWriter(ui.ChatView), "[red]Failed to connect to server: %v\n", err)
        return
    }
    defer conn.Close()

    // Sending username to server
    fmt.Fprintf(conn, "%s\n", username)

    // Setting up message sending functionality
    ui.InputField.SetDoneFunc(func(key tcell.Key) {
        if key == tcell.KeyEnter {
            message := ui.InputField.GetText()
            if message != "" {
                fmt.Fprintf(conn, "%s: %s\n", username, message)
                ui.InputField.SetText("")
                
                // Play send message sound ONLY here
                tmpFileName, err := alert.PrepareSoundFile("out.wav")
                if err != nil {
                    fmt.Println("Error preparing send message sound:", err)
                } else {
                    if err := alert.ExecuteSoundPlayback(tmpFileName); err != nil {
                        fmt.Println("Error playing send message sound:", err)
                    }
                }
            }
        }
    })

    // Handling incoming messages
    go func() {
        scanner := bufio.NewScanner(conn)
        for scanner.Scan() {
            text := scanner.Text()

            // Check if the username is already taken
            if strings.HasPrefix(text, "SYSTEM_MESSAGE:UsernameTaken") {
                fmt.Fprintln(tview.ANSIWriter(ui.ChatView), "[red]Username already taken. Please restart the client and choose a different username.[white]")
                time.Sleep(2 * time.Second)
                conn.Close()
                ui.App.Stop()
                return
            }

            // Processing received message
            ui.App.QueueUpdateDraw(func() {
                if strings.HasPrefix(text, "System:") {
                    fmt.Fprintln(tview.ANSIWriter(ui.ChatView), "[yellow]"+text+"[white]")
                } else {
                    parts := strings.SplitN(text, ": ", 2)
                    if len(parts) == 2 && parts[0] != username { // Ensure message is not from the user
                        // Play receive message sound ONLY for messages from others
                        tmpFileName, err := alert.PrepareSoundFile("incoming.wav")
                        if err != nil {
                            fmt.Println("Error preparing receive message sound:", err)
                        } else {
                            if err := alert.ExecuteSoundPlayback(tmpFileName); err != nil {
                                fmt.Println("Error playing receive message sound:", err)
                            }
                        }
                    }

                    // Display message in chat view
                    colorTag := getColorTag(parts[0])
                    fmt.Fprintf(tview.ANSIWriter(ui.ChatView), "[%s]%s:[white] %s\n", colorTag, parts[0], parts[1])
                }
            })
        }
    }()

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




func getColorTag(username string) string {
    // Hash the username to determine its color index.
    h := fnv.New32a()
    h.Write([]byte(username))
    colorIndex := h.Sum32() % uint32(len(colors))

    // Map the hashed color index to a tview color tag.
    colorTags := map[uint32]string{
        0: "red",
        1: "green",
        2: "yellow",
        3: "blue",
        4: "purple",
        5: "cyan",
        6: "white",
    }

    return colorTags[colorIndex]
}