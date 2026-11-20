package services

import (
	"fmt"
	"strings"
	"testing"
)

func TestReportCatalog(t *testing.T) {
	catalog, err := buildCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog) != 10 {
		t.Fatalf("%d entries, want q01-q10", len(catalog))
	}
	for i, e := range catalog {
		if want := fmt.Sprintf("q%02d", i+1); e.ID != want {
			t.Fatalf("entry %d is %s, want %s", i, e.ID, want)
		}
		if e.Name == "" || e.Description == "" || len(e.Endpoints) == 0 || len(e.Queries) == 0 {
			t.Fatalf("%s is incomplete: %+v", e.ID, e)
		}
		for _, q := range e.Queries {
			if !strings.Contains(e.SQL, "-- name: "+q+" ") {
				t.Fatalf("%s: SQL lacks %s", e.ID, q)
			}
		}
	}
	catalogSpec = append(catalogSpec, catalogSpec[0])
	catalogSpec[len(catalogSpec)-1].Queries = []string{"NoSuchQuery"}
	defer func() { catalogSpec = catalogSpec[:len(catalogSpec)-1] }()
	if _, err := buildCatalog(); err == nil {
		t.Fatal("an unknown query name must fail")
	}
}
