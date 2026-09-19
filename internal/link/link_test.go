package link_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yorukot/ground-news-tw/internal/ai"
	"github.com/yorukot/ground-news-tw/internal/db"
	"github.com/yorukot/ground-news-tw/internal/dbtest"
	"github.com/yorukot/ground-news-tw/internal/link"
)

// fixture inserts analyzed articles the way the summarize job leaves them.
type fixture struct {
	t      *testing.T
	pool   *pgxpool.Pool
	q      *db.Queries
	outlet int64
	n      int
}

type article struct {
	development string
	recaps      []string
	entities    []string
	reprintOf   int64
	ago         time.Duration
}

func newFixture(t *testing.T) *fixture {
	pool := dbtest.Open(t)
	q := db.New(pool)
	outlet, err := q.UpsertOutlet(context.Background(), db.UpsertOutletParams{Slug: "test", Name: "Test", Domain: "example.com", CrawlConfig: []byte("{}"), Coverage: "full"})
	if err != nil {
		t.Fatal(err)
	}
	return &fixture{t: t, pool: pool, q: q, outlet: outlet}
}

func (f *fixture) add(a article) int64 {
	f.t.Helper()
	ctx := context.Background()
	f.n++
	id, err := f.q.InsertFetchedArticle(ctx, db.InsertFetchedArticleParams{
		OutletID:    f.outlet,
		Url:         fmt.Sprintf("https://example.com/%d", f.n),
		Headline:    fmt.Sprintf("headline %d", f.n),
		Body:        "body",
		PublishedAt: time.Now().Add(-a.ago),
		ContentHash: fmt.Sprint(f.n),
		Minhash:     []int32{},
	})
	if err != nil {
		f.t.Fatal(err)
	}
	if a.reprintOf != 0 {
		if err := f.q.SetArticleReprint(ctx, db.SetArticleReprintParams{ID: id, ReprintOfID: pgtype.Int8{Int64: a.reprintOf, Valid: true}}); err != nil {
			f.t.Fatal(err)
		}
		return id
	}
	if err := f.q.SetArticleAnalysis(ctx, db.SetArticleAnalysisParams{ID: id, Development: a.development}); err != nil {
		f.t.Fatal(err)
	}
	recaps := a.recaps
	if recaps == nil {
		recaps = []string{}
	}
	if err := f.q.UpsertSummary(ctx, db.UpsertSummaryParams{ArticleID: id, Text: "A summary of article " + fmt.Sprint(f.n), Recaps: recaps, Model: "fake", PromptVersion: ai.PromptVersion}); err != nil {
		f.t.Fatal(err)
	}
	for _, name := range a.entities {
		// ON CONFLICT makes this return the existing entity for a repeated name.
		eid, err := f.q.CreateEntity(ctx, db.CreateEntityParams{CanonicalName: name, Kind: "person", Aliases: []string{}, SearchText: name})
		if err != nil {
			f.t.Fatal(err)
		}
		if err := f.q.LinkArticleEntity(ctx, db.LinkArticleEntityParams{ArticleID: id, EntityID: eid}); err != nil {
			f.t.Fatal(err)
		}
	}
	return id
}

func (f *fixture) link(id int64) link.Result {
	f.t.Helper()
	res, err := (&link.Linker{Pool: f.pool, AI: ai.Fake{}}).Link(context.Background(), id)
	if err != nil {
		f.t.Fatalf("link %d: %v", id, err)
	}
	return res
}

func TestFollowUpJoinsItsEventHoweverOld(t *testing.T) {
	f := newFixture(t)

	arrest := f.add(article{development: "Police arrest Wang on suspicion of drug possession", entities: []string{"王小明"}, ago: 400 * 24 * time.Hour})
	first := f.link(arrest)
	if !first.NewEvent || !first.NewStep {
		t.Fatalf("first article: %+v, want a new event with a new step", first)
	}

	// More than a year later, a follow-up that recaps the arrest.
	verdict := f.add(article{
		development: "The district court convicts Wang at first instance",
		recaps:      []string{"Police arrest Wang on suspicion of drug possession"},
		entities:    []string{"王小明"},
	})
	second := f.link(verdict)
	if second.NewEvent || second.EventID != first.EventID || !second.NewStep {
		t.Fatalf("follow-up: %+v, want event %d with a new step", second, first.EventID)
	}

	// Another outlet reporting the same verdict attaches to the existing step.
	sameVerdict := f.add(article{development: "The district court convicts Wang at first instance", entities: []string{"王小明"}})
	third := f.link(sameVerdict)
	if third.EventID != first.EventID || third.NewStep {
		t.Fatalf("second report of a step: %+v, want event %d and no new step", third, first.EventID)
	}

	steps, err := f.q.ListEventSteps(context.Background(), []int64{first.EventID})
	if err != nil {
		t.Fatal(err)
	}
	if len(steps) != 2 || steps[0].ID != arrest || steps[1].ID != verdict {
		t.Fatalf("timeline = %+v, want the arrest then the verdict", steps)
	}

	// The event's search text follows its timeline, so later articles find it.
	found, err := f.q.SearchEventCandidates(context.Background(), db.SearchEventCandidatesParams{SearchText: "court convicts", MaxResults: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].ID != first.EventID {
		t.Fatalf("search = %+v, want event %d", found, first.EventID)
	}
}

func TestSharedNameAloneNeverMergesEvents(t *testing.T) {
	f := newFixture(t)
	first := f.link(f.add(article{development: "Mayor Chen opens a new metro line", entities: []string{"陳市長"}}))

	// Same person, unrelated story: the event is nominated as a candidate,
	// but nothing connects the stories, so it must not be merged.
	other := f.link(f.add(article{development: "Mayor Chen is questioned in a land deal investigation", entities: []string{"陳市長"}}))
	if !other.NewEvent || other.EventID == first.EventID {
		t.Fatalf("unrelated story: %+v, want its own event", other)
	}
}

func TestLinkIsIdempotentAndSkipsReprints(t *testing.T) {
	f := newFixture(t)
	original := f.add(article{development: "The council passes the budget"})
	first := f.link(original)

	again := f.link(original)
	if again.Skipped == "" || again.EventID != first.EventID {
		t.Fatalf("relink: %+v, want it skipped as already linked", again)
	}

	reprint := f.add(article{reprintOf: original})
	if res := f.link(reprint); res.Skipped != "reprint" {
		t.Fatalf("reprint: %+v, want it skipped", res)
	}
}

func TestReprintsFollowTheirOriginalIntoItsEvent(t *testing.T) {
	f := newFixture(t)
	original := f.add(article{development: "The weather agency issues a land typhoon warning"})
	reprint := f.add(article{reprintOf: original}) // deduped before the original was linked
	res := f.link(original)

	got, err := f.q.GetArticleForLink(context.Background(), reprint)
	if err != nil {
		t.Fatal(err)
	}
	if !got.EventID.Valid || got.EventID.Int64 != res.EventID {
		t.Fatalf("reprint event = %+v, want %d", got.EventID, res.EventID)
	}
}
