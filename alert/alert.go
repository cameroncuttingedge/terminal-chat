package alert

import (
	"embed"
	"errors"
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

//go:embed icon/clock.png
var clockPNG embed.FS

func EndOfTimer(soundFilePath, title, message string) {
	// Play the end of timer sound in a non-blocking way
	go func() {
		tmpFileName, err := PrepareSoundFile(soundFilePath)
		if err != nil {
			log.Printf("Error preparing sound file: %v", err)
			return
		}

		// This doesn't always work as someimes the applciation quits beforehand
		// See the cleanup func in main.go for the backup plan
		defer func() {
			if removeErr := os.Remove(tmpFileName); removeErr != nil {
				log.Printf("Error removing temporary file '%s': %v", tmpFileName, removeErr)
			}
		}()

		// Play the sound
		err = ExecuteSoundPlayback(tmpFileName)
		if err != nil {
			log.Printf("Error playing sound: %v", err)
		}
	}()

	// Execute notification display in a separate goroutine
	go func() {
		err := ShowNotification(title, message)
		if err != nil {
			log.Printf("Error showing notification: %v", err)
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
	clockFile, err := clockPNG.Open("icon/clock.png")
	if err != nil {
		log.Printf("Error opening embedded image 'WAV/clock.png': %v", err)
		return err
	}
	defer clockFile.Close()

	// Create a temporary file for the image
	tmpFile, err := os.CreateTemp("", "clock-*.png")
	if err != nil {
		log.Printf("Error creating temporary file for image: %v", err)
		return err
	}
	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name()) // Clean up the temp file after use

	// Copy the clock image content to the temporary file
	_, err = io.Copy(tmpFile, clockFile)
	if err != nil {
		log.Printf("Error copying image to temporary file '%s': %v", tmpFile.Name(), err)
		return err
	}

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
