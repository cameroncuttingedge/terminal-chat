package alert

import (
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/cameroncuttingedge/terminal-chat/util"
	"github.com/gen2brain/beeep"
)

//go:embed WAV/*
var wavFS embed.FS

func PlaySoundAsync(fileName string) {
	go func() {
		tmpFileName, err := PrepareSoundFile(fileName)
		if err != nil {
			fmt.Printf("Error preparing send message sound: %v\n", err)
			return
		}
		defer os.Remove(tmpFileName) // Ensure the file is cleaned up after playing

		if err := ExecuteSoundPlayback(tmpFileName); err != nil {
			fmt.Printf("Error playing send message sound: %v\n", err)
		}
	}()
}

func ExecuteSoundPlayback(tmpFileName string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		if util.CmdExists("afplay") {
			cmd = exec.Command("afplay", tmpFileName)
		} else {
			return errors.New("no compatible media player found")
		}
	case "linux":
		if util.CmdExists("ffplay") {
			cmd = exec.Command("ffplay", "-nodisp", "-autoexit", tmpFileName)
		} else if util.CmdExists("mpg123") {
			cmd = exec.Command("mpg123", tmpFileName)
		} else if util.CmdExists("paplay") {
			cmd = exec.Command("paplay", tmpFileName)
		} else if util.CmdExists("aplay") {
			cmd = exec.Command("aplay", tmpFileName)
		} else {
			return errors.New("no compatible media player found")
		}
	case "windows":
		if util.CmdExists("powershell") {
			cmdStr := `$player = New-Object System.Media.SoundPlayer;` +
				`$player.SoundLocation = '` + tmpFileName + `';` +
				`$player.PlaySync();`
			cmd = exec.Command("powershell", "-Command", cmdStr)
		} else {
			return errors.New("no compatible media player found")
		}
	default:
		return errors.New("unsupported platform")
	}

	return cmd.Run()
}

func PrepareSoundFile(filePath string) (string, error) {
	// Generate a new temporary file name
	util.GenerateTempSoundFileName()
	tmpFileName := util.TempFileName // Use the generated temporary file name

	soundFilePath := "WAV/" + filePath // Ensure this path is correct for your application
	soundFile, err := wavFS.Open(soundFilePath)
	if err != nil {
		log.Printf("Error opening embedded sound file '%s': %v", soundFilePath, err)
		return "", err
	}
	defer soundFile.Close()

	// Open the temporary file for writing
	tmpFile, err := os.Create(tmpFileName)
	if err != nil {
		log.Printf("Error creating temporary file '%s': %v", tmpFileName, err)
		return "", err
	}
	defer tmpFile.Close()

	// Copy the sound data to the temporary file
	if _, err := io.Copy(tmpFile, soundFile); err != nil {
		log.Printf("Error copying to temporary file '%s': %v", tmpFileName, err)
		return "", err
	}

	return tmpFileName, nil
}

func ShowNotification(title, message string) error {
	// Create a temporary file for the image
	tmpFile, err := os.CreateTemp("", "clock-*.png")
	if err != nil {
		log.Printf("Error creating temporary file for image: %v", err)
		return err
	}
	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name()) // Clean up the temp file after use

	// Use the path of the temp file for the icon in beeep.Notify
	iconPath := tmpFile.Name()
	err = beeep.Notify(title, message, iconPath)
	if err != nil {
		log.Printf("Error showing notification: %v", err)
		return err
	}
	return nil
}

func ListValidSounds() ([]string, error) {
	var sounds []string

	// Read the directory from the embedded filesystem
	err := fs.WalkDir(wavFS, "WAV", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			sounds = append(sounds, filepath.Base(path))
		}
		return nil
	})

	return sounds, err
}
