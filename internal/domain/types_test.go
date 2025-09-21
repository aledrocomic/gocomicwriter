package domain

import (
	"encoding/json"
	"testing"
)

func TestProjectJSONRoundTrip(t *testing.T) {
	p := Project{
		Name: "RoundTrip",
		Issues: []Issue{
			{
				TrimWidth:        210,
				TrimHeight:       297,
				Bleed:            3,
				DPI:              300,
				ReadingDirection: "ltr",
				Pages: []Page{
					{Number: 1, Grid: "3x3", Panels: []Panel{}},
				},
			},
		},
	}

	b, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got Project
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Name != p.Name {
		t.Fatalf("name mismatch: got %q want %q", got.Name, p.Name)
	}
	if len(got.Issues) != 1 || len(got.Issues[0].Pages) != 1 {
		t.Fatalf("unexpected issues/pages structure: %+v", got)
	}
}
