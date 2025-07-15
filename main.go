package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	fyneDialog "fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
	"github.com/sqweek/dialog"
)

// cleanField cleans a string by removing unescaped double quotes
// and replacing escaped double quotes with a single one.
// It also returns a boolean indicating if a change was made.
func cleanField(text string) (string, bool) {
	originalText := text
	const placeholder = "||--DOUBLE_QUOTE--||"
	text = strings.ReplaceAll(text, `""`, placeholder)
	text = strings.ReplaceAll(text, `"`, "")
	text = strings.ReplaceAll(text, placeholder, `"`)
	cleanedText := strings.TrimSpace(text)
	return cleanedText, cleanedText != originalText
}

// processFile reads the input file, cleans the first column,
// and writes the result to an output file. It returns the number of lines processed and cleaned.
func processFile(inputFilename string, onComplete func(processed, cleaned int)) {
	var processedCount, cleanedCount int
	defer func() {
		onComplete(processedCount, cleanedCount)
	}()

	base := filepath.Base(inputFilename)
	outFilename := fmt.Sprintf("cleaned_%s", base)

	inputFile, err := os.Open(inputFilename)
	if err != nil {
		log.Printf("Не удалось открыть исходный файл: %s", err)
		return
	}
	defer inputFile.Close()

	outFile, err := os.Create(outFilename)
	if err != nil {
		log.Printf("Не удалось создать файл для записи: %s", err)
		return
	}
	defer outFile.Close()

	outWriter := bufio.NewWriter(outFile)
	defer outWriter.Flush()

	scanner := bufio.NewScanner(inputFile)
	for scanner.Scan() {
		processedCount++
		line := scanner.Text()
		parts := strings.SplitN(line, "	", 2)
		firstField := parts[0]

		cleaned, wasCleaned := cleanField(firstField)
		if wasCleaned {
			cleanedCount++
		}

		if _, err := outWriter.WriteString(cleaned + "\n"); err != nil {
			log.Printf("Не удалось записать в файл: %s", err)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Ошибка при чтении исходного файла: %s", err)
	}
}

// validateAndGetInfo checks the file format and returns metadata.
func validateAndGetInfo(uri fyne.URI) (string, string, int, error) {
	file, err := os.Open(uri.Path())
	if err != nil {
		return "", "", 0, err
	}
	defer file.Close()

	// Check header/first line
	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return "", "", 0, fmt.Errorf("файл пуст")
	}

	line := scanner.Text()
	parts := strings.Split(line, "	")
	if len(parts) < 3 {
		return "", "", 0, fmt.Errorf("неверный формат файла (требуется 3 колонки, разделенные табом)")
	}

	gtin := parts[1]
	title := parts[2]

	// Count lines
	file.Seek(0, 0)
	lineCount := 0
	scanner = bufio.NewScanner(file)
	for scanner.Scan() {
		lineCount++
	}

	return gtin, title, lineCount, nil
}

func main() {
	a := app.NewWithID("com.example.diversey")
	w := a.NewWindow("Конвертер файлов")

	var selectedFile fyne.URI

	// --- UI Elements ---
	infoContainer := container.NewVBox()
	infoContainer.Hide()

	gtinLabel := widget.NewLabel("")
	titleLabel := widget.NewLabel("")
	linesLabel := widget.NewLabel("")

	infoBox := container.NewVBox(
		widget.NewLabel("Информация о файле:"),
		container.NewHBox(widget.NewLabel("GTIN:"), gtinLabel),
		container.NewHBox(widget.NewLabel("Название:"), titleLabel),
		container.NewHBox(widget.NewLabel("Количество строк:"), linesLabel),
	)

	progressBar := widget.NewProgressBarInfinite()
	progressBar.Hide()

	convertButton := widget.NewButton("Конвертировать", func() {
		if selectedFile == nil {
			fyneDialog.ShowInformation("Ошибка", "Пожалуйста, выберите файл для конвертации", w)
			return
		}
		progressBar.Show()
		go processFile(selectedFile.Path(), func(processed, cleaned int) {
			fyne.Do(func() {
				progressBar.Hide()
			})

			stats := fmt.Sprintf("Обработано строк: %d\nОчищено строк: %d", processed, cleaned)
			fyneDialog.ShowInformation("Завершено", stats, w)
		})
	})
	convertButton.Disable()

	selectFileButton := widget.NewButton("Выбрать файл", func() {
		filename, err := dialog.File().Filter("CSV и текстовые файлы", "csv", "txt").Load()
		if err != nil {
			if err != dialog.ErrCancelled {
				fyneDialog.ShowError(err, w)
			}
			return
		}

		selectedFile, err = storage.ParseURI("file://" + filename)
		if err != nil {
			if err != dialog.ErrCancelled {
				fyneDialog.ShowError(err, w)
			}
			return
		}
		gtin, title, lineCount, err := validateAndGetInfo(selectedFile)
		if err != nil {
			fyneDialog.ShowError(err, w)
			infoContainer.Hide()
			convertButton.Disable()
			return
		}

		gtinLabel.SetText(gtin)
		titleLabel.SetText(title)
		linesLabel.SetText(fmt.Sprintf("%d", lineCount))
		infoContainer.Show()
		convertButton.Enable()
	})

	w.SetContent(container.NewVBox(
		selectFileButton,
		infoBox,
		progressBar,
		convertButton,
	))

	w.Resize(fyne.NewSize(500, 250))
	w.ShowAndRun()
}
