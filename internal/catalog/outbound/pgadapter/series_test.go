package pgadapter_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/JLugagne/podcast/internal/catalog/domain"
	"github.com/JLugagne/podcast/internal/catalog/domain/repositories/seriesrepo"
	"github.com/JLugagne/podcast/internal/catalog/outbound/pgadapter"
)

func TestSeriesCreateAndGet(t *testing.T) {
	pool := requireDB(t)
	repo := pgadapter.NewSeriesRepository(pool)
	ctx := context.Background()

	s := mustSeriesAt(t, "my-show", "My Show", testNow)
	s.SetCover(domain.Asset{Key: "series/x/cover-1.png", ContentType: "image/png", SizeBytes: 42}, testNow)
	if err := repo.Create(ctx, s); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.Get(ctx, s.ID())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID() != s.ID() || got.Slug() != "my-show" || got.Title() != "My Show" {
		t.Errorf("Get returned %+v, want the created series", got.Attributes())
	}
	if got.Author() != "Ada Lovelace" || got.Description() != "A show about numbers" || got.AccentColor() != "#123456" {
		t.Errorf("attributes not stored: %+v", got.Attributes())
	}
	if got.Version() != 0 {
		t.Errorf("Version = %d, want 0 after Create", got.Version())
	}
	if !got.CreatedAt().Equal(testNow) || !got.UpdatedAt().Equal(testNow) {
		t.Errorf("timestamps = (%s, %s), want %s", got.CreatedAt(), got.UpdatedAt(), testNow)
	}
	cover := got.Cover()
	if cover == nil || cover.Key != "series/x/cover-1.png" || cover.ContentType != "image/png" || cover.SizeBytes != 42 {
		t.Errorf("Cover = %+v, want the stored asset", cover)
	}

	bySlug, err := repo.GetBySlug(ctx, "my-show")
	if err != nil {
		t.Fatalf("GetBySlug: %v", err)
	}
	if bySlug.ID() != s.ID() {
		t.Errorf("GetBySlug id = %s, want %s", bySlug.ID(), s.ID())
	}
}

func TestSeriesNotFound(t *testing.T) {
	pool := requireDB(t)
	repo := pgadapter.NewSeriesRepository(pool)
	ctx := context.Background()

	missing := domain.SeriesID(domain.NewID())
	if _, err := repo.Get(ctx, missing); !errors.Is(err, domain.ErrSeriesNotFound) {
		t.Errorf("Get missing = %v, want ErrSeriesNotFound", err)
	}
	if _, err := repo.GetBySlug(ctx, "nope"); !errors.Is(err, domain.ErrSeriesNotFound) {
		t.Errorf("GetBySlug missing = %v, want ErrSeriesNotFound", err)
	}
	if err := repo.Delete(ctx, missing); !errors.Is(err, domain.ErrSeriesNotFound) {
		t.Errorf("Delete missing = %v, want ErrSeriesNotFound", err)
	}
}

func TestSeriesSlugConflict(t *testing.T) {
	pool := requireDB(t)
	repo := pgadapter.NewSeriesRepository(pool)
	ctx := context.Background()

	if err := repo.Create(ctx, mustSeriesAt(t, "same-slug", "First", testNow)); err != nil {
		t.Fatalf("Create first: %v", err)
	}
	err := repo.Create(ctx, mustSeriesAt(t, "same-slug", "Second", testNow))
	if !errors.Is(err, domain.ErrSlugTaken) {
		t.Fatalf("Create duplicate slug = %v, want ErrSlugTaken", err)
	}

	first, err := repo.GetBySlug(ctx, "same-slug")
	if err != nil {
		t.Fatalf("GetBySlug: %v", err)
	}
	second := mustSeriesAt(t, "other-slug", "Third", testNow)
	if err := repo.Create(ctx, second); err != nil {
		t.Fatalf("Create second: %v", err)
	}
	if err := second.Update(seriesAttrs("same-slug", "Third"), testNow); err != nil {
		t.Fatalf("Update entity: %v", err)
	}
	if err := repo.Update(ctx, second); !errors.Is(err, domain.ErrSlugTaken) {
		t.Fatalf("Update to a taken slug = %v, want ErrSlugTaken", err)
	}
	if first.Slug() != "same-slug" {
		t.Errorf("first slug changed to %q", first.Slug())
	}
}

func TestSeriesUpdateIncrementsVersion(t *testing.T) {
	pool := requireDB(t)
	repo := pgadapter.NewSeriesRepository(pool)
	ctx := context.Background()

	s := mustSeriesAt(t, "my-show", "My Show", testNow)
	if err := repo.Create(ctx, s); err != nil {
		t.Fatalf("Create: %v", err)
	}
	loaded, err := repo.Get(ctx, s.ID())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	later := testNow.Add(time.Hour)
	if err := loaded.Update(seriesAttrs("my-show", "Renamed Show"), later); err != nil {
		t.Fatalf("Update entity: %v", err)
	}
	loaded.ClearCover(later)
	if err := repo.Update(ctx, loaded); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if loaded.Version() != 1 {
		t.Errorf("aggregate Version = %d, want 1", loaded.Version())
	}

	got, err := repo.Get(ctx, s.ID())
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if got.Title() != "Renamed Show" {
		t.Errorf("Title = %q, want Renamed Show", got.Title())
	}
	if got.Version() != 1 {
		t.Errorf("stored Version = %d, want 1", got.Version())
	}
	if !got.UpdatedAt().Equal(later) {
		t.Errorf("UpdatedAt = %s, want %s", got.UpdatedAt(), later)
	}
	if got.HasCover() {
		t.Errorf("Cover = %+v, want nil after ClearCover", got.Cover())
	}
}

func TestSeriesUpdateConflict(t *testing.T) {
	pool := requireDB(t)
	repo := pgadapter.NewSeriesRepository(pool)
	ctx := context.Background()

	s := mustSeriesAt(t, "my-show", "My Show", testNow)
	if err := repo.Create(ctx, s); err != nil {
		t.Fatalf("Create: %v", err)
	}

	first, err := repo.Get(ctx, s.ID())
	if err != nil {
		t.Fatalf("Get first: %v", err)
	}
	second, err := repo.Get(ctx, s.ID())
	if err != nil {
		t.Fatalf("Get second: %v", err)
	}

	if err := first.Update(seriesAttrs("my-show", "First writer"), testNow); err != nil {
		t.Fatalf("Update first entity: %v", err)
	}
	if err := repo.Update(ctx, first); err != nil {
		t.Fatalf("Update first: %v", err)
	}

	if err := second.Update(seriesAttrs("my-show", "Second writer"), testNow); err != nil {
		t.Fatalf("Update second entity: %v", err)
	}
	if err := repo.Update(ctx, second); !errors.Is(err, domain.ErrVersionConflict) {
		t.Fatalf("stale Update = %v, want ErrVersionConflict", err)
	}

	got, err := repo.Get(ctx, s.ID())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Title() != "First writer" {
		t.Errorf("Title = %q, want First writer", got.Title())
	}
}

func TestSeriesUpdateMissing(t *testing.T) {
	pool := requireDB(t)
	repo := pgadapter.NewSeriesRepository(pool)
	ctx := context.Background()

	ghost := domain.ReconstituteSeries(domain.SeriesID(domain.NewID()), seriesAttrs("ghost", "Ghost"), nil, testNow, testNow, 0)
	if err := repo.Update(ctx, ghost); !errors.Is(err, domain.ErrSeriesNotFound) {
		t.Fatalf("Update missing = %v, want ErrSeriesNotFound", err)
	}
}

func TestSeriesList(t *testing.T) {
	pool := requireDB(t)
	repo := pgadapter.NewSeriesRepository(pool)
	ctx := context.Background()

	series := []*domain.Series{
		mustSeriesAt(t, "zeta", "Zeta Show", testNow.Add(-2*time.Hour)),
		mustSeriesAt(t, "alpha", "Alpha Show", testNow.Add(-time.Hour)),
		mustSeriesAt(t, "beta", "Beta Show", testNow),
	}
	for _, s := range series {
		if err := repo.Create(ctx, s); err != nil {
			t.Fatalf("Create %s: %v", s.Slug(), err)
		}
	}

	all, err := repo.List(ctx, seriesrepo.Filter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	wantOrder := []string{"beta", "alpha", "zeta"}
	if len(all) != len(wantOrder) {
		t.Fatalf("List returned %d series, want %d", len(all), len(wantOrder))
	}
	for i, want := range wantOrder {
		if all[i].Slug() != want {
			t.Errorf("List[%d].Slug = %q, want %q", i, all[i].Slug(), want)
		}
	}

	filtered, err := repo.List(ctx, seriesrepo.Filter{Query: "BETA"})
	if err != nil {
		t.Fatalf("List with query: %v", err)
	}
	if len(filtered) != 1 || filtered[0].Slug() != "beta" {
		t.Errorf("List(query=beta) = %v, want only beta", slugs(filtered))
	}

	byAuthor, err := repo.List(ctx, seriesrepo.Filter{Query: "lovelace"})
	if err != nil {
		t.Fatalf("List by author: %v", err)
	}
	if len(byAuthor) != 3 {
		t.Errorf("List(query=lovelace) returned %d series, want 3", len(byAuthor))
	}

	wildcard, err := repo.List(ctx, seriesrepo.Filter{Query: "%"})
	if err != nil {
		t.Fatalf("List with wildcard: %v", err)
	}
	if len(wildcard) != 0 {
		t.Errorf("List(query=%%)= %v, want no match (the wildcard must be escaped)", slugs(wildcard))
	}

	limited, err := repo.List(ctx, seriesrepo.Filter{Limit: 2, Offset: 1})
	if err != nil {
		t.Fatalf("List paginated: %v", err)
	}
	if len(limited) != 2 || limited[0].Slug() != "alpha" || limited[1].Slug() != "zeta" {
		t.Errorf("List(limit=2, offset=1) = %v, want [alpha zeta]", slugs(limited))
	}
}

func TestSeriesDelete(t *testing.T) {
	pool := requireDB(t)
	seriesRepo := pgadapter.NewSeriesRepository(pool)
	episodeRepo := pgadapter.NewEpisodeRepository(pool)
	ctx := context.Background()

	s := mustSeriesAt(t, "my-show", "My Show", testNow)
	if err := seriesRepo.Create(ctx, s); err != nil {
		t.Fatalf("Create series: %v", err)
	}
	e := mustEpisodeAt(t, s.ID(), 1, "Pilot", testNow)
	if err := episodeRepo.Create(ctx, e); err != nil {
		t.Fatalf("Create episode: %v", err)
	}

	if err := seriesRepo.Delete(ctx, s.ID()); !errors.Is(err, domain.ErrSeriesNotEmpty) {
		t.Fatalf("Delete series with episodes = %v, want ErrSeriesNotEmpty", err)
	}
	if err := episodeRepo.Delete(ctx, e.ID()); err != nil {
		t.Fatalf("Delete episode: %v", err)
	}
	if err := seriesRepo.Delete(ctx, s.ID()); err != nil {
		t.Fatalf("Delete series: %v", err)
	}
	if _, err := seriesRepo.Get(ctx, s.ID()); !errors.Is(err, domain.ErrSeriesNotFound) {
		t.Errorf("Get after Delete = %v, want ErrSeriesNotFound", err)
	}
}

func slugs(series []*domain.Series) []string {
	out := make([]string, len(series))
	for i, s := range series {
		out[i] = s.Slug()
	}
	return out
}
