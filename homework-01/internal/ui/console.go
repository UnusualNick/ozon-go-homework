package ui

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/UnusualNick/ozon-go-homework/homework-01/internal/business"
	"github.com/UnusualNick/ozon-go-homework/homework-01/internal/domain"
)

const help = `commands:
  help
  add <title> <deadline>
  list [--status STATUS] [--search TEXT] [--page N] [--limit N]
  get <id>
  status <id> <planned|in_progress|canceled|done>
  edit <id> [--title TITLE] [--deadline DEADLINE]
  delete <id>
  import <path>
  exit

deadline formats: YYYY-MM-DD or RFC3339
put arguments containing spaces in quotes`

type Console struct {
	service *business.Service
	in      io.Reader
	out     io.Writer
	errOut  io.Writer
}

func NewConsole(service *business.Service, in io.Reader, out, errOut io.Writer) *Console {
	return &Console{service: service, in: in, out: out, errOut: errOut}
}

func (c *Console) Run() error {
	scanner := bufio.NewScanner(c.in)
	for {
		fmt.Fprint(c.out, "> ")
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("read command: %w", err)
			}
			return nil
		}
		arguments, err := splitArguments(scanner.Text())
		if err != nil {
			c.printError(err)
			continue
		}
		if len(arguments) == 0 {
			continue
		}
		if arguments[0] == "exit" {
			if len(arguments) != 1 {
				c.printError(fmt.Errorf("usage: exit"))
				continue
			}
			return nil
		}
		if err := c.execute(arguments); err != nil {
			c.printError(err)
		}
	}
}

func (c *Console) execute(arguments []string) error {
	switch arguments[0] {
	case "help":
		if len(arguments) != 1 {
			return fmt.Errorf("usage: help")
		}
		fmt.Fprintln(c.out, help)
		return nil
	case "add":
		return c.add(arguments[1:])
	case "list":
		return c.list(arguments[1:])
	case "get":
		return c.get(arguments[1:])
	case "status":
		return c.status(arguments[1:])
	case "edit":
		return c.edit(arguments[1:])
	case "delete":
		return c.delete(arguments[1:])
	case "import":
		return c.importTasks(arguments[1:])
	default:
		return fmt.Errorf("unknown command %q; use help", arguments[0])
	}
}

func (c *Console) add(arguments []string) error {
	if len(arguments) != 2 {
		return fmt.Errorf("usage: add <title> <deadline>")
	}
	deadline, err := parseDeadline(arguments[1])
	if err != nil {
		return err
	}
	task, err := c.service.Add(arguments[0], deadline)
	if err != nil {
		return err
	}
	fmt.Fprintf(c.out, "created task %d\n", task.ID)
	return nil
}

func (c *Console) list(arguments []string) error {
	options := business.ListOptions{}
	for index := 0; index < len(arguments); index++ {
		if index+1 >= len(arguments) {
			return fmt.Errorf("option %s requires a value", arguments[index])
		}
		option, value := arguments[index], arguments[index+1]
		index++
		switch option {
		case "--status":
			status, err := domain.ParseStatus(value)
			if err != nil {
				return err
			}
			options.Status = &status
		case "--search":
			options.Search = value
		case "--page":
			page, err := parsePositiveInt(value, "page")
			if err != nil {
				return err
			}
			options.Page = page
		case "--limit":
			limit, err := parsePositiveInt(value, "limit")
			if err != nil {
				return err
			}
			options.Limit = limit
		default:
			return fmt.Errorf("unknown list option %q", option)
		}
	}
	tasks, err := c.service.List(options)
	if err != nil {
		return err
	}
	if len(tasks) == 0 {
		fmt.Fprintln(c.out, "no tasks")
		return nil
	}
	for _, task := range tasks {
		c.printTask(task)
	}
	return nil
}

func (c *Console) get(arguments []string) error {
	if len(arguments) != 1 {
		return fmt.Errorf("usage: get <id>")
	}
	id, err := parseID(arguments[0])
	if err != nil {
		return err
	}
	task, err := c.service.Get(id)
	if err != nil {
		return err
	}
	c.printTask(task)
	return nil
}

func (c *Console) status(arguments []string) error {
	if len(arguments) != 2 {
		return fmt.Errorf("usage: status <id> <status>")
	}
	id, err := parseID(arguments[0])
	if err != nil {
		return err
	}
	status, err := domain.ParseStatus(arguments[1])
	if err != nil {
		return err
	}
	task, err := c.service.SetStatus(id, status)
	if err != nil {
		return err
	}
	fmt.Fprintf(c.out, "updated task %d\n", task.ID)
	return nil
}

func (c *Console) edit(arguments []string) error {
	if len(arguments) < 3 {
		return fmt.Errorf("usage: edit <id> [--title TITLE] [--deadline DEADLINE]")
	}
	id, err := parseID(arguments[0])
	if err != nil {
		return err
	}
	request := business.EditTask{}
	for index := 1; index < len(arguments); index++ {
		if index+1 >= len(arguments) {
			return fmt.Errorf("option %s requires a value", arguments[index])
		}
		option, value := arguments[index], arguments[index+1]
		index++
		switch option {
		case "--title":
			request.Title = &value
		case "--deadline":
			deadline, err := parseDeadline(value)
			if err != nil {
				return err
			}
			request.Deadline = &deadline
		default:
			return fmt.Errorf("unknown edit option %q", option)
		}
	}
	task, err := c.service.Edit(id, request)
	if err != nil {
		return err
	}
	fmt.Fprintf(c.out, "updated task %d\n", task.ID)
	return nil
}

func (c *Console) delete(arguments []string) error {
	if len(arguments) != 1 {
		return fmt.Errorf("usage: delete <id>")
	}
	id, err := parseID(arguments[0])
	if err != nil {
		return err
	}
	if err := c.service.Delete(id); err != nil {
		return err
	}
	fmt.Fprintf(c.out, "deleted task %d\n", id)
	return nil
}

func (c *Console) importTasks(arguments []string) error {
	if len(arguments) != 1 {
		return fmt.Errorf("usage: import <path>")
	}
	data, err := os.ReadFile(arguments[0])
	if err != nil {
		return fmt.Errorf("read import file: %w", err)
	}
	var tasks []domain.Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		return fmt.Errorf("decode import file: %w", err)
	}
	count, err := c.service.Import(tasks)
	if err != nil {
		return err
	}
	fmt.Fprintf(c.out, "imported %d task(s)\n", count)
	return nil
}

func (c *Console) printTask(task domain.Task) {
	fmt.Fprintf(c.out, "%d | %s | %s | %s\n", task.ID, task.Deadline.Format(time.RFC3339), task.Status, task.Title)
}

func (c *Console) printError(err error) {
	if business.IsNotFound(err) {
		fmt.Fprintf(c.errOut, "error: %s\n", err)
		return
	}
	fmt.Fprintf(c.errOut, "error: %s\n", err)
}

func parseID(value string) (int64, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("id must be a positive integer")
	}
	return id, nil
}

func parsePositiveInt(value, name string) (int, error) {
	number, err := strconv.Atoi(value)
	if err != nil || number <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	return number, nil
}

func parseDeadline(value string) (time.Time, error) {
	for _, layout := range []string{"2006-01-02", time.RFC3339} {
		deadline, err := time.Parse(layout, value)
		if err == nil {
			return deadline.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("deadline must use YYYY-MM-DD or RFC3339 format")
}

func splitArguments(line string) ([]string, error) {
	var arguments []string
	var current strings.Builder
	var quote rune
	escaped := false
	started := false

	flush := func() {
		if started {
			arguments = append(arguments, current.String())
			current.Reset()
			started = false
		}
	}

	for _, character := range line {
		if escaped {
			current.WriteRune(character)
			escaped = false
			started = true
			continue
		}
		if character == '\\' {
			escaped = true
			started = true
			continue
		}
		if quote != 0 {
			if character == quote {
				quote = 0
			} else {
				current.WriteRune(character)
			}
			started = true
			continue
		}
		if character == '\'' || character == '"' {
			quote = character
			started = true
			continue
		}
		if unicode.IsSpace(character) {
			flush()
			continue
		}
		current.WriteRune(character)
		started = true
	}
	if escaped {
		return nil, errors.New("unfinished escape sequence")
	}
	if quote != 0 {
		return nil, errors.New("unterminated quote")
	}
	flush()
	return arguments, nil
}
