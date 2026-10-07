package jsonfile

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/UnusualNick/ozon-go-homework/homework-01/internal/domain"
)

func TestRepositoryPersistsTasks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "tasks.json")
	repository := New(path)
	deadline, err := time.Parse(time.RFC3339, "2026-10-20T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	want := []domain.Task{{ID: 1, Title: "task", Status: domain.StatusPlanned, Deadline: deadline}}

	if err := repository.Save(want); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := repository.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("loaded %#v, want %#v", got, want)
	}
}

func TestRepositoryReturnsEmptyForMissingFile(t *testing.T) {
	repository := New(filepath.Join(t.TempDir(), "missing.json"))
	tasks, err := repository.Load()
	if err != nil {
		t.Fatal(err)
	}
	if tasks == nil || len(tasks) != 0 {
		t.Fatalf("got %#v, want non-nil empty slice", tasks)
	}
}

func TestRepositoryRejectsInvalidPersistedTask(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")
	if err := os.WriteFile(path, []byte(`[{"id":1,"title":"","status":"planned","deadline":"2026-10-20T00:00:00Z"}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := New(path).Load()
	if err == nil {
		t.Fatal("expected validation error")
	}
}
