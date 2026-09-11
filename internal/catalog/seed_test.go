package catalog

import (
	"path/filepath"
	"strings"
	"testing"

	"dndmanager/internal/platform"
)

func TestPHBSeedAndWizardOptions(t *testing.T) {
	dir := t.TempDir()
	db, err := platform.OpenDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	mig := filepath.Join("..", "..", "migrations")
	if err := platform.Migrate(db, mig); err != nil {
		t.Fatal(err)
	}

	svc := &Service{Repo: &Repository{DB: db}}
	races, err := svc.ListRaces()
	if err != nil {
		t.Fatal(err)
	}
	bgs, err := svc.ListBackgrounds()
	if err != nil {
		t.Fatal(err)
	}
	classes, err := svc.ListClasses()
	if err != nil {
		t.Fatal(err)
	}
	items, err := svc.ListItems()
	if err != nil {
		t.Fatal(err)
	}
	spells, err := svc.ListSpells()
	if err != nil {
		t.Fatal(err)
	}

	if len(races) != 11 {
		t.Fatalf("races %d want 11", len(races))
	}
	if len(bgs) != 13 {
		t.Fatalf("backgrounds %d want 13", len(bgs))
	}
	if len(classes) != 12 {
		t.Fatalf("classes %d want 12", len(classes))
	}
	if len(items) != 9 {
		t.Fatalf("items %d want 9", len(items))
	}
	if len(spells) != 8 {
		t.Fatalf("spells %d want 8", len(spells))
	}

	var raceCount, classCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM catalog_subclasses`).Scan(&classCount); err != nil {
		t.Fatal(err)
	}
	if classCount != 15 {
		t.Fatalf("subclasses %d want 15", classCount)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM catalog_features`).Scan(&raceCount); err != nil {
		t.Fatal(err)
	}
	if raceCount < 90 {
		t.Fatalf("features %d want >= 90", raceCount)
	}

	for _, r := range races {
		if r.SourceURLRU == "" || !strings.Contains(r.SourceURLRU, "5e14.dnd.su/race/") {
			t.Fatalf("race %s missing 5e14 URL: %q", r.Slug, r.SourceURLRU)
		}
		if strings.Contains(r.SourceURLRU, "/race/human/") || strings.Contains(r.SourceURLRU, "dnd.su/race/elf/") {
			t.Fatalf("race %s still has invented slug path %s", r.Slug, r.SourceURLRU)
		}
	}
	for _, c := range classes {
		if !strings.Contains(c.SourceURLRU, "5e14.dnd.su/class/") || !strings.Contains(c.SourceURLRU, "-") {
			t.Fatalf("class %s bad 5e14 URL %s", c.Slug, c.SourceURLRU)
		}
		if c.SourceURL == "" || !strings.HasPrefix(c.SourceURL, "https://www.dnd5eapi.co/api/2014/classes/") {
			t.Fatalf("class %s bad 5eapi URL %s", c.Slug, c.SourceURL)
		}
	}

	warlock, err := svc.Class(12)
	if err != nil || warlock.Slug != "warlock" || warlock.SubclassLevel != 1 {
		t.Fatalf("warlock row: %+v %v", warlock, err)
	}
	subs, err := svc.ListSubclasses(12)
	if err != nil || len(subs) == 0 || subs[0].Slug != "fiend" {
		t.Fatalf("warlock subclasses: %+v %v", subs, err)
	}
	if !strings.Contains(warlock.SourceURLRU, "104-warlock") {
		t.Fatalf("warlock RU URL %s", warlock.SourceURLRU)
	}
}
