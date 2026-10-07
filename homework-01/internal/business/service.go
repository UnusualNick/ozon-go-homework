package business

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/UnusualNick/ozon-go-homework/homework-01/internal/domain"
)

// Repository is declared by the business layer that consumes it.
type Repository interface {
	Load() ([]domain.Task, error)
	Save([]domain.Task) error
}

type NotFoundError struct {
	ID int64
}

func (e *NotFoundError) Error() string { return fmt.Sprintf("task %d not found", e.ID) }

type ConflictError struct {
	ID int64
}

func (e *ConflictError) Error() string { return fmt.Sprintf("task %d already exists", e.ID) }

func IsNotFound(err error) bool {
	var target *NotFoundError
	return errors.As(err, &target)
}

type ListOptions struct {
	Status *domain.Status
	Search string
	Page   int
	Limit  int
}

type EditTask struct {
	Title    *string
	Deadline *time.Time
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Add(title string, deadline time.Time) (domain.Task, error) {
	tasks, err := s.load()
	if err != nil {
		return domain.Task{}, err
	}

	task := domain.Task{
		ID:       nextID(tasks),
		Title:    strings.TrimSpace(title),
		Status:   domain.StatusPlanned,
		Deadline: deadline.UTC(),
	}
	if err := task.Validate(); err != nil {
		return domain.Task{}, fmt.Errorf("validate task: %w", err)
	}

	tasks = append(tasks, task)
	if err := s.save(tasks); err != nil {
		return domain.Task{}, err
	}
	return task, nil
}

func (s *Service) List(options ListOptions) ([]domain.Task, error) {
	tasks, err := s.load()
	if err != nil {
		return nil, err
	}

	query := strings.ToLower(strings.TrimSpace(options.Search))
	filtered := make([]domain.Task, 0, len(tasks))
	for _, task := range tasks {
		if options.Status != nil && task.Status != *options.Status {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(task.Title), query) {
			continue
		}
		filtered = append(filtered, task)
	}
	sortTasks(filtered)

	if options.Page < 0 {
		return nil, fmt.Errorf("page must be positive")
	}
	if options.Limit < 0 {
		return nil, fmt.Errorf("limit must be positive")
	}
	if options.Page == 0 && options.Limit == 0 {
		return filtered, nil
	}
	page, limit := options.Page, options.Limit
	if page == 0 {
		page = 1
	}
	if limit == 0 {
		limit = 10
	}
	start := (page - 1) * limit
	if start >= len(filtered) {
		return []domain.Task{}, nil
	}
	end := min(start+limit, len(filtered))
	return filtered[start:end], nil
}

func (s *Service) Get(id int64) (domain.Task, error) {
	tasks, err := s.load()
	if err != nil {
		return domain.Task{}, err
	}
	index := findByID(tasks, id)
	if index < 0 {
		return domain.Task{}, &NotFoundError{ID: id}
	}
	return tasks[index], nil
}

func (s *Service) SetStatus(id int64, status domain.Status) (domain.Task, error) {
	if !status.Valid() {
		return domain.Task{}, fmt.Errorf("validate status: invalid status %q", status)
	}
	tasks, err := s.load()
	if err != nil {
		return domain.Task{}, err
	}
	index := findByID(tasks, id)
	if index < 0 {
		return domain.Task{}, &NotFoundError{ID: id}
	}
	tasks[index].Status = status
	if err := s.save(tasks); err != nil {
		return domain.Task{}, err
	}
	return tasks[index], nil
}

func (s *Service) Edit(id int64, edit EditTask) (domain.Task, error) {
	if edit.Title == nil && edit.Deadline == nil {
		return domain.Task{}, fmt.Errorf("at least one field must be specified")
	}
	tasks, err := s.load()
	if err != nil {
		return domain.Task{}, err
	}
	index := findByID(tasks, id)
	if index < 0 {
		return domain.Task{}, &NotFoundError{ID: id}
	}
	if edit.Title != nil {
		tasks[index].Title = strings.TrimSpace(*edit.Title)
	}
	if edit.Deadline != nil {
		tasks[index].Deadline = edit.Deadline.UTC()
	}
	if err := tasks[index].Validate(); err != nil {
		return domain.Task{}, fmt.Errorf("validate task: %w", err)
	}
	if err := s.save(tasks); err != nil {
		return domain.Task{}, err
	}
	return tasks[index], nil
}

func (s *Service) Delete(id int64) error {
	tasks, err := s.load()
	if err != nil {
		return err
	}
	index := findByID(tasks, id)
	if index < 0 {
		return &NotFoundError{ID: id}
	}
	tasks = append(tasks[:index], tasks[index+1:]...)
	return s.save(tasks)
}

func (s *Service) Import(imported []domain.Task) (int, error) {
	tasks, err := s.load()
	if err != nil {
		return 0, err
	}
	existing := make(map[int64]struct{}, len(tasks)+len(imported))
	for _, task := range tasks {
		existing[task.ID] = struct{}{}
	}

	next := nextID(tasks)
	prepared := make([]domain.Task, 0, len(imported))
	for index, task := range imported {
		if task.ID == 0 {
			for {
				if _, exists := existing[next]; !exists {
					break
				}
				next++
			}
			task.ID = next
			next++
		}
		if _, exists := existing[task.ID]; exists {
			return 0, fmt.Errorf("import item %d: %w", index+1, &ConflictError{ID: task.ID})
		}
		if task.Status == "" {
			task.Status = domain.StatusPlanned
		}
		task.Title = strings.TrimSpace(task.Title)
		task.Deadline = task.Deadline.UTC()
		if err := task.Validate(); err != nil {
			return 0, fmt.Errorf("import item %d: %w", index+1, err)
		}
		existing[task.ID] = struct{}{}
		prepared = append(prepared, task)
	}

	tasks = append(tasks, prepared...)
	if err := s.save(tasks); err != nil {
		return 0, err
	}
	return len(prepared), nil
}

func (s *Service) load() ([]domain.Task, error) {
	tasks, err := s.repository.Load()
	if err != nil {
		return nil, fmt.Errorf("load tasks: %w", err)
	}
	return tasks, nil
}

func (s *Service) save(tasks []domain.Task) error {
	if err := s.repository.Save(tasks); err != nil {
		return fmt.Errorf("save tasks: %w", err)
	}
	return nil
}

func nextID(tasks []domain.Task) int64 {
	var maximum int64
	for _, task := range tasks {
		if task.ID > maximum {
			maximum = task.ID
		}
	}
	return maximum + 1
}

func findByID(tasks []domain.Task, id int64) int {
	for index, task := range tasks {
		if task.ID == id {
			return index
		}
	}
	return -1
}

func sortTasks(tasks []domain.Task) {
	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].Deadline.Equal(tasks[j].Deadline) {
			return tasks[i].ID < tasks[j].ID
		}
		return tasks[i].Deadline.Before(tasks[j].Deadline)
	})
}
