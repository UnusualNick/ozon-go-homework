package business

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/UnusualNick/ozon-go-homework/homework-01/internal/domain"
)

type memoryRepository struct {
	tasks   []domain.Task
	loadErr error
	saveErr error
}

func (r *memoryRepository) Load() ([]domain.Task, error) {
	if r.loadErr != nil {
		return nil, r.loadErr
	}
	return append([]domain.Task(nil), r.tasks...), nil
}

func (r *memoryRepository) Save(tasks []domain.Task) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	r.tasks = append([]domain.Task(nil), tasks...)
	return nil
}

func mustTime(t *testing.T, value string) time.Time {
	t.Helper()
	result, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestServiceCRUDAndStableIDs(t *testing.T) {
	repository := &memoryRepository{}
	service := NewService(repository)

	first, err := service.Add("  first  ", mustTime(t, "2026-10-20T00:00:00Z"))
	if err != nil {
		t.Fatalf("add first: %v", err)
	}
	second, err := service.Add("second", mustTime(t, "2026-10-21T00:00:00Z"))
	if err != nil {
		t.Fatalf("add second: %v", err)
	}
	if first.ID != 1 || second.ID != 2 || first.Title != "first" {
		t.Fatalf("unexpected added tasks: %#v %#v", first, second)
	}

	statusTask, err := service.SetStatus(first.ID, domain.StatusDone)
	if err != nil {
		t.Fatalf("set status: %v", err)
	}
	newTitle := "renamed"
	edited, err := service.Edit(first.ID, EditTask{Title: &newTitle})
	if err != nil {
		t.Fatalf("edit: %v", err)
	}
	if statusTask.Status != domain.StatusDone || edited.Title != newTitle {
		t.Fatalf("updates were not applied: %#v %#v", statusTask, edited)
	}

	if err := service.Delete(second.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	third, err := service.Add("third", mustTime(t, "2026-10-22T00:00:00Z"))
	if err != nil {
		t.Fatalf("add after deletion: %v", err)
	}
	if third.ID != 2 {
		t.Fatalf("got id %d, want 2 as the next free id", third.ID)
	}
}

func TestListSortFilterSearchAndPagination(t *testing.T) {
	deadline := mustTime(t, "2026-10-20T00:00:00Z")
	repository := &memoryRepository{tasks: []domain.Task{
		{ID: 3, Title: "Write report", Status: domain.StatusPlanned, Deadline: deadline},
		{ID: 2, Title: "Another", Status: domain.StatusDone, Deadline: deadline.Add(-time.Hour)},
		{ID: 1, Title: "REPORT review", Status: domain.StatusPlanned, Deadline: deadline},
	}}
	service := NewService(repository)

	all, err := service.List(ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	ids := []int64{all[0].ID, all[1].ID, all[2].ID}
	if !reflect.DeepEqual(ids, []int64{2, 1, 3}) {
		t.Fatalf("sort order = %v, want [2 1 3]", ids)
	}

	status := domain.StatusPlanned
	page, err := service.List(ListOptions{Status: &status, Search: "report", Page: 2, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != 1 || page[0].ID != 3 {
		t.Fatalf("unexpected filtered page: %#v", page)
	}
}

func TestImportIsAtomicAndAssignsIDs(t *testing.T) {
	repository := &memoryRepository{tasks: []domain.Task{{
		ID: 1, Title: "existing", Status: domain.StatusPlanned,
		Deadline: mustTime(t, "2026-10-20T00:00:00Z"),
	}}}
	service := NewService(repository)

	count, err := service.Import([]domain.Task{{
		Title: "new", Deadline: mustTime(t, "2026-10-21T00:00:00Z"),
	}})
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if count != 1 || len(repository.tasks) != 2 || repository.tasks[1].ID != 2 || repository.tasks[1].Status != domain.StatusPlanned {
		t.Fatalf("unexpected imported tasks: %#v", repository.tasks)
	}

	before := append([]domain.Task(nil), repository.tasks...)
	_, err = service.Import([]domain.Task{
		{ID: 3, Title: "valid", Status: domain.StatusDone, Deadline: mustTime(t, "2026-10-22T00:00:00Z")},
		{ID: 1, Title: "duplicate", Status: domain.StatusDone, Deadline: mustTime(t, "2026-10-23T00:00:00Z")},
	})
	if err == nil {
		t.Fatal("expected duplicate id error")
	}
	if !reflect.DeepEqual(repository.tasks, before) {
		t.Fatalf("failed import changed repository: %#v", repository.tasks)
	}
}

func TestRepositoryErrorsAreWrapped(t *testing.T) {
	sentinel := errors.New("disk failed")
	service := NewService(&memoryRepository{loadErr: sentinel})
	_, err := service.Get(1)
	if !errors.Is(err, sentinel) {
		t.Fatalf("load error was not wrapped: %v", err)
	}
}

func TestNotFoundUsesBusinessError(t *testing.T) {
	service := NewService(&memoryRepository{})
	_, err := service.Get(42)
	if !IsNotFound(err) {
		t.Fatalf("expected NotFoundError, got %v", err)
	}
}
