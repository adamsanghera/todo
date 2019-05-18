package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	todoRoot  = "/Users/adam.sanghera/.todo"
	threshold = 7
)

type ongoing struct {
	Created      time.Time
	DisappearsAt time.Time
	Content      string
}

type memory interface {
	Render() string
	CreatedAt() time.Time
	Append(content string)
	AppearsToday() bool
}

func (o *ongoing) CreatedAt() time.Time  { return o.Created }
func (o *ongoing) Append(content string) { o.Content += "\n" + content }
func (o *ongoing) AppearsToday() bool    { return o.DisappearsAt.After(time.Now()) }
func (o *ongoing) Render() string {
	if len(strings.Split(o.Content, "\n")) > 1 {
		memory := "\n"
		for _, line := range strings.Split(o.Content, "\n") {
			memory += fmt.Sprintf("    > %s\n", line)
		}
		memory += "\n"
		return memory
	}
	return fmt.Sprintf("  - %s\n", o.Content)
}

func todoFile(date time.Time, flag int) (*os.File, error) {
	fpath := filepath.Join(todoRoot, date.Format("2006-01-02")+".md")
	return os.OpenFile(fpath, flag, 0777)
}

func parseMemories(memSrc io.Reader, sourceTime time.Time) []memory {
	/*
	 * Long memories are block quotes embedded in the list of short memories.
	 * Short memories are single-line elements in a list.
	 *
	 * They end up formatted together like this:
	 *
	 * - 2019-05-15
	 *   - short memory
	 *   - short memory
	 *
	 *     > long memory start
	 *     > long memory middle
	 *     > long memory end
	 *
	 *   - short memory
	 *   - short memory
	 *
	 */
	startsM := func(text string) (bool, int) {
		if strings.HasPrefix(text, "```") && strings.Contains(text, ".remember-for=") {
			// try to find a day count string
			numberHalf := strings.Split(text, ".remember-for=")[1]
			var numberDays int
			for idx := 1; idx < len(numberHalf); idx++ {
				candidate, err := strconv.Atoi(numberHalf[:idx])
				if err != nil {
					break
				}
				numberDays = candidate
			}
			return true, numberDays
		}
		return false, 0
	}
	endsM := func(text string) bool {
		return strings.TrimSpace(text) == "```"
	}

	memories := make([]memory, 0)
	inMemory := false
	scn := bufio.NewScanner(memSrc)
	for scn.Scan() {
		text := scn.Text()

		// Decide whether this line is important
		if inMemory && endsM(text) {
			inMemory = !inMemory
		}

		// Decide to add text or not
		if inMemory {
			memories[len(memories)-1].Append(text)
		} else if strings.HasPrefix(text, "/remember=") {
			//	daysToRemember :=
			numberHalf := strings.Split(text, "=")[1]
			var numberDays int
			for idx := 1; idx < len(numberHalf); idx++ {
				candidate, err := strconv.Atoi(numberHalf[:idx])
				if err != nil {
					break
				}
				numberDays = candidate
			}
			memories = append(memories, &ongoing{
				Content: strings.TrimSpace(
					text[len("/remember="+strconv.Itoa(numberDays)):]),
				Created:      sourceTime,
				DisappearsAt: sourceTime.AddDate(0, 0, numberDays),
			})
		}

		// Decide whether next line is important
		if !inMemory {
			if starts, daysToRemember := startsM(text); starts {
				inMemory = true
				memories = append(memories, &ongoing{
					Content:      "",
					Created:      sourceTime,
					DisappearsAt: sourceTime.AddDate(0, 0, daysToRemember),
				})
			}
		}
	}
	return memories
}

func collectMemoriesThreshold() (string, error) {
	memories := ""
	for idx := 1; idx < threshold; idx++ {
		// get the previous day's file
		prevDayF, err := todoFile(time.Now().AddDate(0, 0, -idx), os.O_RDONLY)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", err
		}

		// The file opened OK, so we scan it
		memories += fmt.Sprintf("- %s\n", time.Now().AddDate(0, 0, -idx).Format("2006-01-02"))
		rawMemories := parseMemories(prevDayF, time.Now().AddDate(0, 0, -idx))
		for _, rawMemory := range rawMemories {
			if rawMemory.AppearsToday() {
				memories += rawMemory.Render()
			}
		}
	}
	return memories, nil
}

// RootCmd is the entrypoint
var RootCmd = cobra.Command{
	Use:   "todo",
	Short: "a self-contained cli for managing TODO's",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create the directory if it does not exist
		err := os.Mkdir(todoRoot, 0777)
		if err != nil && os.IsNotExist(err) {
			return err
		}

		// Open today's file (creating it if it doesn't exist)
		todayFile, err := todoFile(time.Now(), os.O_RDONLY)
		if err != nil {
			if !os.IsNotExist(err) {
				return err
			}

			// The file did not exist before, so we have to create + fill it

			// create the file
			todayFile, err = todoFile(time.Now(), os.O_CREATE|os.O_RDWR)
			if err != nil {
				return err
			}

			// collect memories from recent days
			memories, err := collectMemoriesThreshold()
			if err != nil {
				return err
			}
			if memories == "" {
				memories = "...nothing to see here...\n"
			}

			// write memories + template
			_, err = todayFile.WriteString(
				fmt.Sprintf("# Memories\n\n%s\n# On the past\n\n\n# On today\n\n\n", memories))
			if err != nil {
				return err
			}
		}

		// the file exists and is filled
		err = todayFile.Close()
		if err != nil {
			return err
		}

		// Open editor
		ed := exec.Command(os.Getenv("EDITOR"), todayFile.Name())
		ed.Stdin = os.Stdin
		ed.Stdout = os.Stdout
		ed.Stderr = os.Stderr

		return ed.Run()
	},
}

func main() {
	err := RootCmd.Execute()
	if err != nil {
		log.Fatal(err)
	}
}
