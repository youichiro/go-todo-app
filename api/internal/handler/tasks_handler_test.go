package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"

	"github.com/youichiro/go-todo-app/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
)

var cmpOption cmp.Option

func TestMain(m *testing.M) {
	setup()
	status := m.Run()
	teardown()
	os.Exit(status)
}

func setup() {
	fmt.Println("setup")
	cmpOption = cmpopts.IgnoreFields(models.Task{}, "CreatedAt", "UpdatedAt")
}

func teardown() {
	fmt.Println("teardown")
}

// func initMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
// 	mockDB, mock, err := sqlmock.New()
// 	assert.NoError(t, err)
// 	client.DB = mockDB
// 	return mockDB, mock
// }

func createTestContext(method string, path string, jsonString string) (*httptest.ResponseRecorder, *gin.Context) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	if jsonString != "" {
		c.Request = httptest.NewRequest(method, path, bytes.NewBuffer([]byte(jsonString)))
	} else {
		c.Request = httptest.NewRequest(method, path, nil)
	}
	c.Request.Header.Set("Content-Type", "application/json")

	return w, c
}

func TestTaskHandlerIndex(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		mockDB, mock, err := sqlmock.New()
		assert.NoError(t, err)

		rows := mock.NewRows([]string{"id", "title", "done"})
		rows.AddRow(0, "dummy task1", false)
		rows.AddRow(1, "dummy task2", true)
		query := regexp.QuoteMeta(`SELECT "tasks".* FROM "tasks"`)
		mock.ExpectQuery(query).WillReturnRows(rows)

		w, c := createTestContext("GET", "/tasks", "")
		TaskHander{DB: mockDB}.Index(c)

		assert.Equal(t, 200, w.Code)

		var tasks []models.Task
		body, _ := io.ReadAll(w.Body)
		err = json.Unmarshal(body, &tasks)
		assert.NoError(t, err)

		expectBodyFirst := models.Task{ID: 0, Title: "dummy task1", Done: false}
		expectBodySecond := models.Task{ID: 1, Title: "dummy task2", Done: true}
		assert.Equal(t, 2, len(tasks))
		assert.Empty(t, cmp.Diff(expectBodyFirst, tasks[0], cmpOption))
		assert.Empty(t, cmp.Diff(expectBodySecond, tasks[1], cmpOption))

		t.Cleanup(func() {
			mockDB.Close()
		})
	})

	t.Run("異常系_SELECTに失敗する場合", func(t *testing.T) {
		t.Parallel()

		mockDB, mock, err := sqlmock.New()
		assert.NoError(t, err)

		query := regexp.QuoteMeta(`SELECT "tasks".* FROM "tasks"`)
		mock.ExpectQuery(query).WillReturnError(fmt.Errorf("error"))

		w, c := createTestContext("GET", "/tasks", "")
		TaskHander{DB: mockDB}.Index(c)

		assert.Equal(t, 404, w.Code)

		t.Cleanup(func() {
			mockDB.Close()
		})
	})
}

func TestTaskHandlerShow(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		mockDB, mock, err := sqlmock.New()
		assert.NoError(t, err)

		rows := mock.NewRows([]string{"id", "title", "done"}).AddRow(3, "dummy task3", false)
		query := regexp.QuoteMeta(`select * from "tasks" where "id"=$1`)
		mock.ExpectQuery(query).WillReturnRows(rows)

		w, c := createTestContext("GET", "/tasks/3", "")
		TaskHander{DB: mockDB}.Show(c)

		assert.Equal(t, 200, w.Code)

		var task models.Task
		body, _ := io.ReadAll(w.Body)
		err = json.Unmarshal(body, &task)
		assert.NoError(t, err)

		expectBody := models.Task{ID: 3, Title: "dummy task3", Done: false}
		assert.Empty(t, cmp.Diff(expectBody, task, cmpOption))

		t.Cleanup(func() {
			mockDB.Close()
		})
	})

	t.Run("異常系_レコードが存在しない場合", func(t *testing.T) {
		t.Parallel()

		mockDB, mock, err := sqlmock.New()
		assert.NoError(t, err)

		query := regexp.QuoteMeta(`select * from "tasks" where "id"=$1`)
		mock.ExpectQuery(query).WillReturnError(fmt.Errorf("error"))

		w, c := createTestContext("GET", "/tasks/3", "")
		TaskHander{DB: mockDB}.Show(c)

		assert.Equal(t, 404, w.Code)

		t.Cleanup(func() {
			mockDB.Close()
		})
	})
}

func TestTaskHandlerCreate(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		mockDB, mock, err := sqlmock.New()
		assert.NoError(t, err)

		rows := mock.NewRows([]string{"id", "done"}).AddRow(0, false)
		query := regexp.QuoteMeta(`INSERT INTO "tasks" ("title","created_at","updated_at") VALUES ($1,$2,$3) RETURNING "id","done"`)
		mock.ExpectQuery(query).WillReturnRows(rows)

		w, c := createTestContext("POST", "/tasks", `{"title": "dummy insert task"}`)
		TaskHander{DB: mockDB}.Create(c)

		assert.Equal(t, 201, w.Code)

		var task models.Task
		body, _ := io.ReadAll(w.Body)
		err = json.Unmarshal(body, &task)
		assert.NoError(t, err)
		expectBody := models.Task{ID: 0, Title: "dummy insert task", Done: false}
		assert.Empty(t, cmp.Diff(expectBody, task, cmpOption))

		t.Cleanup(func() {
			mockDB.Close()
		})
	})

	t.Run("異常系_INSERTに失敗した場合", func(t *testing.T) {
		t.Parallel()

		mockDB, mock, err := sqlmock.New()
		assert.NoError(t, err)

		query := regexp.QuoteMeta(`INSERT INTO "tasks" ("title","created_at","updated_at") VALUES ($1,$2,$3) RETURNING "id","done"`)
		mock.ExpectQuery(query).WillReturnError(fmt.Errorf("error"))

		w, c := createTestContext("POST", "/tasks", `{"title": "dummy insert task"}`)
		TaskHander{DB: mockDB}.Create(c)

		assert.Equal(t, 500, w.Code)

		t.Cleanup(func() {
			mockDB.Close()
		})
	})

	t.Run("異常系_リクエストパラメーターが間違えている場合", func(t *testing.T) {
		t.Parallel()

		mockDB, _, err := sqlmock.New()
		assert.NoError(t, err)

		w, c := createTestContext("POST", "/tasks", `{"invalid_title": "invalid task"}`)
		TaskHander{DB: mockDB}.Create(c)

		assert.Equal(t, 400, w.Code)

		t.Cleanup(func() {
			mockDB.Close()
		})
	})
}

func TestTaskHandlerUpdate(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		mockDB, mock, err := sqlmock.New()
		assert.NoError(t, err)

		// mock find query
		rows := mock.NewRows([]string{"id", "title", "done"}).AddRow(1, "dummy task", false)
		query := regexp.QuoteMeta(`select * from "tasks" where "id"=$1`)
		mock.ExpectQuery(query).WillReturnRows(rows)

		// mock update exec
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "tasks" SET "title"=$1,"done"=$2,"updated_at"=$3 WHERE "id"=$4`)).
			WillReturnResult(sqlmock.NewResult(1, 1))

		w, c := createTestContext("PUT", "/tasks/1", `{"title": "dummy update task", "done": true}`)
		TaskHander{DB: mockDB}.Update(c)

		assert.Equal(t, 200, w.Code)

		t.Cleanup(func() {
			mockDB.Close()
		})
	})

	t.Run("異常系_UPDATEに失敗した場合", func(t *testing.T) {
		t.Parallel()

		mockDB, mock, err := sqlmock.New()
		assert.NoError(t, err)

		// mock find query
		rows := mock.NewRows([]string{"id", "title", "done"}).AddRow(1, "dummy task", false)
		query := regexp.QuoteMeta(`select * from "tasks" where "id"=$1`)
		mock.ExpectQuery(query).WillReturnRows(rows)

		// mock update exec
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "tasks" SET "title"=$1,"done"=$2,"updated_at"=$3 WHERE "id"=$4`)).
			WillReturnError(fmt.Errorf("error"))

		w, c := createTestContext("PUT", "/tasks/1", `{"title": "dummy insert task", "done": true}`)
		TaskHander{DB: mockDB}.Update(c)

		assert.Equal(t, 500, w.Code)

		t.Cleanup(func() {
			mockDB.Close()
		})
	})

	t.Run("異常系_Findに失敗した場合", func(t *testing.T) {
		t.Parallel()

		mockDB, mock, err := sqlmock.New()
		assert.NoError(t, err)

		// mock find query
		query := regexp.QuoteMeta(`select * from "tasks" where "id"=$1`)
		mock.ExpectQuery(query).WillReturnError(fmt.Errorf("error"))

		w, c := createTestContext("PUT", "/tasks/1", `{"title": "dummy insert task", "done": true}`)
		TaskHander{DB: mockDB}.Update(c)

		assert.Equal(t, 404, w.Code)

		t.Cleanup(func() {
			mockDB.Close()
		})
	})

	t.Run("異常系_リクエストパラメーターが間違えている場合", func(t *testing.T) {
		t.Parallel()

		mockDB, _, err := sqlmock.New()
		assert.NoError(t, err)

		w, c := createTestContext("PUT", "/tasks/1", `{"invalid_title": "invalid task", "done": true}`)
		TaskHander{DB: mockDB}.Update(c)

		assert.Equal(t, 400, w.Code)

		t.Cleanup(func() {
			mockDB.Close()
		})
	})
}

func TestTaskHandlerDelete(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		mockDB, mock, err := sqlmock.New()
		assert.NoError(t, err)

		// mock find query
		rows := mock.NewRows([]string{"id", "title", "done"}).AddRow(1, "dummy task", false)
		query := regexp.QuoteMeta(`select * from "tasks" where "id"=$1`)
		mock.ExpectQuery(query).WillReturnRows(rows)

		// mock delete exec
		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "tasks" WHERE "id"=$1`)).
			WillReturnResult(sqlmock.NewResult(1, 1))

		w, c := createTestContext("DELETE", "/tasks/1", "")
		TaskHander{DB: mockDB}.Delete(c)

		assert.Equal(t, 204, w.Code)

		t.Cleanup(func() {
			mockDB.Close()
		})
	})

	t.Run("異常系_DELETEに失敗した場合", func(t *testing.T) {
		t.Parallel()

		mockDB, mock, err := sqlmock.New()
		assert.NoError(t, err)

		// mock find query
		rows := mock.NewRows([]string{"id", "title", "done"}).AddRow(1, "dummy task", false)
		query := regexp.QuoteMeta(`select * from "tasks" where "id"=$1`)
		mock.ExpectQuery(query).WillReturnRows(rows)

		// mock update exec
		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "tasks" WHERE "id"=$1`)).
			WillReturnError(fmt.Errorf("error"))

		w, c := createTestContext("DELETE", "/tasks/1", "")
		TaskHander{DB: mockDB}.Delete(c)

		assert.Equal(t, 500, w.Code)

		t.Cleanup(func() {
			mockDB.Close()
		})
	})

	t.Run("異常系_Findに失敗した場合", func(t *testing.T) {
		t.Parallel()

		mockDB, mock, err := sqlmock.New()
		assert.NoError(t, err)

		// mock find query
		query := regexp.QuoteMeta(`select * from "tasks" where "id"=$1`)
		mock.ExpectQuery(query).WillReturnError(fmt.Errorf("error"))

		w, c := createTestContext("DELETE", "/tasks/1", "")
		TaskHander{DB: mockDB}.Delete(c)

		assert.Equal(t, 404, w.Code)

		t.Cleanup(func() {
			mockDB.Close()
		})
	})
}
