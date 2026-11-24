package penerimaan

import (
	"context"
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type memoryDistributorStore struct {
	distributors []Distributor
	listErr      error
}

func (s *memoryDistributorStore) List(context.Context) ([]Distributor, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return append([]Distributor(nil), s.distributors...), nil
}

func (s *memoryDistributorStore) Create(_ context.Context, name string) error {
	for _, distributor := range s.distributors {
		if distributor.Name == name {
			return ErrNameTaken
		}
	}
	s.distributors = append(s.distributors, Distributor{ID: len(s.distributors) + 1, Name: name, Active: true})
	return nil
}

func (s *memoryDistributorStore) Rename(_ context.Context, id int, name string) error {
	for i := range s.distributors {
		if s.distributors[i].ID == id {
			for _, distributor := range s.distributors {
				if distributor.ID != id && distributor.Name == name {
					return ErrNameTaken
				}
			}
			s.distributors[i].Name = name
			return nil
		}
	}
	return ErrNotFound
}

func (s *memoryDistributorStore) SetActive(_ context.Context, id int, active bool) error {
	for i := range s.distributors {
		if s.distributors[i].ID == id {
			s.distributors[i].Active = active
			return nil
		}
	}
	return ErrNotFound
}

func testPBFHandler(t *testing.T, store distributorStore) http.Handler {
	t.Helper()
	page, err := template.New("pbf.html").Parse(`{{define "pbf.html"}}{{if .Error}}<p>{{.Error}}</p>{{end}}{{if .NameError}}<p>{{.NameError}}</p>{{end}}{{if .EditError}}<p>{{.EditError}}</p>{{end}}{{if .Empty}}<p>Belum ada distributor PBF.</p>{{end}}{{range .Distributors}}<p>{{.Name}}</p>{{end}}<form method="post" action="/pbf"></form>{{end}}`)
	if err != nil {
		t.Fatal(err)
	}
	return newHandler(store, page)
}

func TestCreateDistributorTrimsNameAndRedirects(t *testing.T) {
	store := &memoryDistributorStore{}
	handler := testPBFHandler(t, store)
	request := httptest.NewRequest(http.MethodPost, "/pbf", strings.NewReader("nama_pbf=+PT+Farmasi+Maju+++"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusSeeOther)
	}
	if response.Header().Get("Location") != "/pbf?status=created" {
		t.Fatalf("Location = %q", response.Header().Get("Location"))
	}
	if len(store.distributors) != 1 || store.distributors[0].Name != "PT Farmasi Maju" || !store.distributors[0].Active {
		t.Fatalf("created distributor = %#v", store.distributors)
	}
}

func TestCreateDistributorRejectsBlankName(t *testing.T) {
	store := &memoryDistributorStore{}
	handler := testPBFHandler(t, store)
	request := httptest.NewRequest(http.MethodPost, "/pbf", strings.NewReader("nama_pbf=++++"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnprocessableEntity)
	}
	if len(store.distributors) != 0 {
		t.Fatalf("blank name created a distributor: %#v", store.distributors)
	}
	if !strings.Contains(response.Body.String(), "Nama distributor wajib diisi.") {
		t.Fatalf("validation message missing from %q", response.Body.String())
	}
}

func TestCreateDistributorRejectsNameLongerThanSchemaLimit(t *testing.T) {
	store := &memoryDistributorStore{}
	handler := testPBFHandler(t, store)
	request := httptest.NewRequest(http.MethodPost, "/pbf", strings.NewReader("nama_pbf="+strings.Repeat("é", 151)))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnprocessableEntity)
	}
	if len(store.distributors) != 0 {
		t.Fatalf("oversized name created a distributor: %#v", store.distributors)
	}
}

func TestCreateDistributorShowsDuplicateNameError(t *testing.T) {
	store := &memoryDistributorStore{distributors: []Distributor{{ID: 1, Name: "PT Farmasi Maju", Active: true}}}
	handler := testPBFHandler(t, store)
	request := httptest.NewRequest(http.MethodPost, "/pbf", strings.NewReader("nama_pbf=PT+Farmasi+Maju"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusConflict)
	}
	if !strings.Contains(response.Body.String(), "Nama distributor sudah terdaftar.") {
		t.Fatalf("duplicate message missing from %q", response.Body.String())
	}
}

func TestEditAndDeactivateDistributor(t *testing.T) {
	store := &memoryDistributorStore{distributors: []Distributor{{ID: 7, Name: "PBF Lama", Active: true}}}
	handler := testPBFHandler(t, store)
	rename := httptest.NewRequest(http.MethodPost, "/pbf/7/edit", strings.NewReader("nama_pbf=PBF+Baru"))
	rename.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	renameResponse := httptest.NewRecorder()
	handler.ServeHTTP(renameResponse, rename)
	if renameResponse.Code != http.StatusSeeOther || store.distributors[0].Name != "PBF Baru" {
		t.Fatalf("rename status = %d, distributor = %#v", renameResponse.Code, store.distributors[0])
	}

	deactivate := httptest.NewRequest(http.MethodPost, "/pbf/7/status", strings.NewReader("aktif=false"))
	deactivate.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	deactivateResponse := httptest.NewRecorder()
	handler.ServeHTTP(deactivateResponse, deactivate)
	if deactivateResponse.Code != http.StatusSeeOther || store.distributors[0].Active {
		t.Fatalf("deactivation status = %d, distributor = %#v", deactivateResponse.Code, store.distributors[0])
	}

	activate := httptest.NewRequest(http.MethodPost, "/pbf/7/status", strings.NewReader("aktif=true"))
	activate.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	activateResponse := httptest.NewRecorder()
	handler.ServeHTTP(activateResponse, activate)
	if activateResponse.Code != http.StatusSeeOther || !store.distributors[0].Active {
		t.Fatalf("activation status = %d, distributor = %#v", activateResponse.Code, store.distributors[0])
	}
}

func TestRenameDistributorShowsInlineValidationAndRetainsEditRow(t *testing.T) {
	store := &memoryDistributorStore{distributors: []Distributor{{ID: 7, Name: "PBF Lama", Active: true}}}
	handler := testPBFHandler(t, store)
	request := httptest.NewRequest(http.MethodPost, "/pbf/7/edit", strings.NewReader("nama_pbf=+++"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnprocessableEntity)
	}
	if !strings.Contains(response.Body.String(), "Nama distributor wajib diisi.") {
		t.Fatalf("edit validation message missing from %q", response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "PBF Lama") {
		t.Fatalf("edit row missing from %q", response.Body.String())
	}
}

func TestListDistributorShowsEmptyAndErrorStates(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		handler := testPBFHandler(t, &memoryDistributorStore{})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/pbf", nil))

		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "Belum ada distributor PBF.") {
			t.Fatalf("empty response = %d, %q", response.Code, response.Body.String())
		}
	})

	t.Run("error", func(t *testing.T) {
		handler := testPBFHandler(t, &memoryDistributorStore{listErr: errors.New("database unavailable")})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/pbf", nil))

		if response.Code != http.StatusInternalServerError || !strings.Contains(response.Body.String(), "Tidak dapat memuat distributor PBF.") {
			t.Fatalf("error response = %d, %q", response.Code, response.Body.String())
		}
	})
}

func TestPBFPageEscapesDistributorNamesAndRendersActions(t *testing.T) {
	page, err := template.ParseFiles("../../templates/penerimaan/pbf.html")
	if err != nil {
		t.Fatal(err)
	}
	store := &memoryDistributorStore{
		distributors: []Distributor{{ID: 11, Name: `<script>alert("x")</script>`, Active: true}},
	}
	handler := newHandler(store, page)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/pbf", nil))

	body := response.Body.String()
	for _, expected := range []string{
		`&lt;script&gt;alert(&#34;x&#34;)&lt;/script&gt;`,
		`action="/pbf/11/edit"`,
		`action="/pbf/11/status"`,
		`aria-current="page">Distributor PBF</a>`,
		`data-bs-toggle="offcanvas"`,
		`/pbf/static/vendor/bootstrap/bootstrap.bundle.min.js`,
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("page does not contain %q", expected)
		}
	}

	if strings.Contains(body, `<td class="pbf-name-cell"><script>`) {
		t.Fatal("distributor name was rendered as markup")
	}
}
