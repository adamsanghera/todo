package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const todoRoot = "/Users/adam.sanghera/.todo"
const threshold = 7

func todoFile(date time.Time, flag int) (*os.File, error) {
	fpath := filepath.Join(todoRoot, date.Format("2006-01-02")+".md")
	return os.OpenFile(fpath, flag, 0777)
}

func startMemory(text string) bool {
	return strings.HasPrefix(text, "```") && strings.Contains(text, ".remember")
}

func endMemory(text string) bool {
	return strings.TrimSpace(text) == "```"
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
			memories := "# Memories\n\n"
			for idx := 1; idx < threshold; idx++ {
				// get the previous day's file
				prevDayF, err := todoFile(time.Now().AddDate(0, 0, -idx), os.O_RDONLY)
				if err != nil {
					if os.IsNotExist(err) {
						continue
					}
					return err
				}

				// The file opened OK, so we scan it for memories

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
				memories += fmt.Sprintf("- %s\n", time.Now().AddDate(0, 0, -idx).Format("2006-01-02"))
				inMemory := false
				scn := bufio.NewScanner(prevDayF)
				for scn.Scan() {
					text := scn.Text()

					// Decide whether this line is important
					if inMemory && endMemory(text) {
						inMemory = !inMemory
						memories += "\n"
					}

					// Decide to add text or not
					if inMemory {
						memories += fmt.Sprintf("    > %s\n", text)
					} else if strings.HasPrefix(text, "/remember") {
						memories += fmt.Sprintf("  - %s\n", text[9:])
					}

					// Decide whether next line is important
					if !inMemory && startMemory(text) {
						inMemory = true
						memories += "\n"
					}
				}
			}
			if memories == "# Memories\n\n" {
				memories += "...nothing to see here...\n"
			}

			// write memories + template
			_, err := todayFile.WriteString(
				fmt.Sprintf("%s\n# On the past\n\n\n# On today\n\n\n", memories))
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
