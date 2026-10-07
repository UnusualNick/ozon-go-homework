package jsonfile

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/UnusualNick/ozon-go-homework/homework-01/internal/domain"
)

type Repository struct {
	path string
}

func New(path string) *Repository {
	return &Repository{path: path}
}

func (r *Repository) Load() ([]domain.Task, error) {
	data, err := os.ReadFile(r.path)
	if errors.Is(err, fs.ErrNotExist) {
		return []domain.Task{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %q: %w", r.path, err)
	}
	if len(data) == 0 {
		return []domain.Task{}, nil
	}
	var tasks []domain.Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		return nil, fmt.Errorf("decode %q: %w", r.path, err)
	}
	if tasks == nil {
		tasks = []domain.Task{}
	}
	for index, task := range tasks {
		if err := task.Validate(); err != nil {
			return nil, fmt.Errorf("decode %q: task %d: %w", r.path, index+1, err)
		}
	}
	return tasks, nil
}

func (r *Repository) Save(tasks []domain.Task) error {
	directory := filepath.Dir(r.path)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create data directory %q: %w", directory, err)
	}
	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return fmt.Errorf("encode tasks: %w", err)
	}
	data = append(data, '\n')

	temporary, err := os.CreateTemp(directory, ".tasks-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	temporaryPath := temporary.Name()
	success := false
	defer func() {
		_ = temporary.Close()
		if !success {
			_ = os.Remove(temporaryPath)
		}
	}()

	if err := temporary.Chmod(0o600); err != nil {
		return fmt.Errorf("set temporary file permissions: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		return fmt.Errorf("write temporary file: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("sync temporary file: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary file: %w", err)
	}
	if err := os.Rename(temporaryPath, r.path); err != nil {
		return fmt.Errorf("replace %q: %w", r.path, err)
	}
	success = true
	return nil
}
