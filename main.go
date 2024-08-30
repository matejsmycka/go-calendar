package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

// Event structure to hold event details
type Event struct {
	UID         string
	Title       string
	Description string
	StartTime   string
	EndTime     string
	Location    string
}

func convertToDateTime(date string) string {
	//format: 20210914T080000

	month := date[4:6]
	day := date[6:8]
	hour := date[9:11]
	minute := date[11:13]
	//output format: Wed 03.12 12:00
	return fmt.Sprintf("%s.%s %s:%s", day, month, hour, minute)

}

func charConvert(data string) string {
	data = strings.ReplaceAll(data, "\\n", "\n")
	data = strings.ReplaceAll(data, "\\t", "\t")
	data = strings.ReplaceAll(data, "\\,", ",")
	return data
}

// Function to parse events from text data
func parseEvents(data string) []Event {
	var events []Event
	var event Event
	scanner := bufio.NewScanner(strings.NewReader(data))
	description := false
	descriptionCount := 0
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		line = charConvert(line)
		if strings.HasPrefix(line, "UID:") {
			event.UID = strings.TrimPrefix(line, "UID:")
			description = false
		} else if strings.HasPrefix(line, "DESCRIPTION:") || description {
			if descriptionCount == 1 {
				// ADJUST HOW MANY LINES OF DESCRIPTION TO DISPLAY
				description = false
				descriptionCount = 0
				event.Description = strings.TrimSuffix(event.Description, "\n")
				event.Description = event.Description + "..."
				continue
			}
			if description {
				event.Description += "\n" + line
				descriptionCount++
				continue
			}

			if strings.HasPrefix(line, "DESCRIPTION:\n") {
				event.Description = "---"
				continue
			}
			event.Description = strings.TrimPrefix(line, "DESCRIPTION:")
			description = true
		} else if strings.HasPrefix(line, "SUMMARY:") {
			event.Title = strings.TrimPrefix(line, "SUMMARY:")
		} else if strings.HasPrefix(line, "DTSTART;TZID=") {
			event.StartTime = strings.Split(strings.TrimPrefix(line, "DTSTART;TZID="), ":")[1]
		} else if strings.HasPrefix(line, "DTEND;TZID=") {
			event.EndTime = strings.Split(strings.TrimPrefix(line, "DTEND;TZID="), ":")[1]
		} else if strings.HasPrefix(line, "LOCATION:") {
			event.Location = strings.TrimPrefix(line, "LOCATION:")
		} else if line == "END:VEVENT" {
			events = append(events, event)
			event = Event{}
		} else {
			description = false
		}

	}
	return events
}

const (
	gray      = lipgloss.Color("245")
	lightGray = lipgloss.Color("241")
	darkGray  = lipgloss.Color("236")
)

func displayEvents(events []Event, limit int) {
	re := lipgloss.NewRenderer(os.Stdout)

	var (
		// HeaderStyle is the lipgloss style used for the table headers.
		HeaderStyle = re.NewStyle().Bold(true).Align(lipgloss.Center)
		// CellStyle is the base lipgloss style used for the table rows.
		CellStyle = re.NewStyle().Padding(1, 1).Width(20)
		// OddRowStyle is the lipgloss style used for odd-numbered table rows.
		OddRowStyle = CellStyle.Foreground(gray).Background(darkGray)
		// EvenRowStyle is the lipgloss style used for even-numbered table rows.
		EvenRowStyle = CellStyle.Foreground(lightGray)
		// BorderStyle is the lipgloss style used for the table border.
		BorderStyle = lipgloss.NewStyle()
	)

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(BorderStyle).
		StyleFunc(func(row, col int) lipgloss.Style {
			var style lipgloss.Style

			switch {
			case row == 0:
				return HeaderStyle
			case row%2 == 0:
				style = EvenRowStyle
			default:
				style = OddRowStyle
			}
			if col == 1 {
				style = style.Width(40).Padding(1, 1)
			}
			if col == 2 || col == 3 {
				style = style.Width(15).Padding(1, 1).Align(lipgloss.Center)
			}
			return style
		}).
		Headers("TITLE", "DESCRIPTION", "START TIME", "END TIME", "LOCATION")

	currentDate := time.Now()

	j := 0
	for _, event := range events {
		j++
		// Skip events that have already ended
		endTime, _ := time.Parse("20060102T150405", event.EndTime)
		if endTime.Before(currentDate) {
			j--
			continue
		}

		if j > limit {
			break
		}

		t.Row(event.Title, event.Description, convertToDateTime(event.StartTime), convertToDateTime(event.EndTime), event.Location)
	}

	fmt.Println(t.Render())
	if j == 0 {
		fmt.Println("No upcoming events")
	}
	if j < limit {
		fmt.Println("Only", j, "events available")
	}
}

func main() {

	url := flag.String("url", "", "URL to download the events")
	limit := flag.Int("limit", 5, "Number of events to display")
	file := flag.String("file", "", "File to read the events from")

	flag.Parse()

	// Perform HTTP GET request
	if *url == "" && *file == "" {
		fmt.Println("Please provide either a URL or a file")
		return

	}

	var events []Event
	var data []byte
	var err error

	if file != nil && *file != "" {
		// Read the file
		data, err = os.ReadFile(*file)
		if err != nil {
			fmt.Println("Failed to read the file:", err)
			return
		}

	} else if url != nil && *url != "" {
		resp, err := http.Get(*url)
		if err != nil {
			fmt.Println("Failed to download the events:", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			fmt.Println("Failed to download the events:", resp.Status)
			return
		}

		data, err = io.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("Failed to read the response body:", err)
			return
		}

	} else {
		fmt.Println("Please provide either a URL or a file")
		return
	}

	events = parseEvents(string(data))
	displayEvents(events, *limit)

}
