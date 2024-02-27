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

var heartbeatChan = make(chan time.Time)

func StartClient() {
	log.Println("Starting client application...")
	serverIP := flag.String("ip", "127.0.0.1", "The IP address of the server to connect to.")
	serverPort := flag.String("port", "9999", "The port of the server to connect to.")

	flag.Parse()

	app := tview.NewApplication()

	// Show login screen and get credentials
	username := showFormScreen(app, "Enter your username", "Username")

	// Ignore for now TODO?
	//password := showFormScreen(app, "Enter your password", "Password")

	// Initialize the UI components
	chatUI := setupUIComponents(app, username)

	// Connect to server and handle chat session
	startChatSession(chatUI, username, *serverIP, *serverPort)
}

func setupUIComponents(app *tview.Application, username string) *ChatUI {
	chatUI := &ChatUI{
		App: app,
	}

	// Initialize the ChatView.
	chatUI.ChatView = tview.NewTextView()
	chatUI.ChatView.SetDynamicColors(true)
	chatUI.ChatView.SetRegions(true)
	chatUI.ChatView.SetScrollable(true)
	chatUI.ChatView.SetBackgroundColor(tcell.ColorDefault)
	chatUI.ChatView.SetBorder(true)
	chatUI.ChatView.SetTitle(" Chat ")
	chatUI.ChatView.SetChangedFunc(func() {
		app.Draw()
	})

	// Initialize the InputField.
	chatUI.InputField = tview.NewInputField()
	chatUI.InputField.SetLabel(fmt.Sprintf("[red]%s[-]: ", username))
	chatUI.InputField.SetFieldWidth(0)
	chatUI.InputField.SetFieldBackgroundColor(tcell.ColorDefault)
	chatUI.InputField.SetBorder(true)
	chatUI.InputField.SetTitle(" Input ")
	chatUI.InputField.SetLabelColor(tcell.ColorDefault)

	// Setup the UI layout with both components.
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(chatUI.ChatView, 0, 1, false).
		AddItem(chatUI.InputField, 3, 1, true)

	app.SetRoot(flex, true).SetFocus(chatUI.InputField)

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			// Switch focus between InputField and ChatView.
			if app.GetFocus() == chatUI.InputField {
				app.SetFocus(chatUI.ChatView)
			} else {
				app.SetFocus(chatUI.InputField)
			}
			return nil
		} else if event.Key() == tcell.KeyBacktab { // Reverse cycling with Shift+Tab.
			if app.GetFocus() == chatUI.InputField {
				app.SetFocus(chatUI.ChatView)
			} else {
				app.SetFocus(chatUI.InputField)
			}
			return nil
		}
		return event
	})

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
		if strings.Contains(text, "SYSTEM_MESSAGE:PING") {
			heartbeatChan <- time.Now()
			continue
		}
		if strings.HasPrefix(text, "SYSTEM_MESSAGE:Color:") {
			parts := strings.Split(text, ":")
			if len(parts) == 3 {
				userColor := parts[2]
				ui.InputField.SetLabel(fmt.Sprintf("%s%s[-]: ", userColor, username))
				continue
			}
		}
		ui.App.QueueUpdateDraw(func() {
			parts := strings.SplitN(text, ": ", 2)
			if len(parts) == 2 && !isUsernameContained(parts[0], username) {
				alert.PlaySoundAsync("in.wav")
				//alert.ShowNotification(parts[0], text)
			}
			fmt.Fprintln(tview.ANSIWriter(ui.ChatView), text)
			ui.ChatView.ScrollToEnd()
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

	// check on server health
	go monitorServerHeartbeat(heartbeatChan, ui)

	// Handling incoming messages
	go handleIncomingMessages(conn, ui, username)

	// Running the tview application
	if err := ui.App.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running application: %v\n", err)
		os.Exit(1)
	}
}

func monitorServerHeartbeat(heartbeatChan <-chan time.Time, ui *ChatUI) {
	timeoutDuration := 30 * time.Second
	heartbeatTimer := time.NewTimer(timeoutDuration)

	for {
		select {
		case <-heartbeatTimer.C: // Timer expired
			fmt.Println("Server connection lost. Shutting down...")
			time.Sleep(3 * time.Second)
			ui.App.Stop()
			fmt.Println("Server connection lost. Shutting down...")
			return
		case heartbeat := <-heartbeatChan: // Received heartbeat
			heartbeatTimer.Reset(timeoutDuration)
			_ = heartbeat
		}
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
	// Regex to find and remove color encoding
	re := regexp.MustCompile(`\[[^\[\]]*\]`)
	cleanStr := re.ReplaceAllString(encodedStr, "")
	return strings.Contains(cleanStr, username)
}
