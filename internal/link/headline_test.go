package link_test

import (
	"context"
	"testing"
	"time"

	"github.com/yorukot/ground-news-tw/internal/db"
	"github.com/yorukot/ground-news-tw/internal/link"
)

func (f *fixture) headline(text string) int64 {
	f.t.Helper()
	f.n++
	id, err := f.q.InsertHeadlineArticle(context.Background(), db.InsertHeadlineArticleParams{
		OutletID: f.outlet, Url: "https://example.com/h/" + text, Headline: text, PublishedAt: time.Now(), ContentHash: text,
	})
	if err != nil {
		f.t.Fatal(err)
	}
	return id
}

func (f *fixture) match(id int64, now time.Time) int64 {
	f.t.Helper()
	eventID, err := link.MatchHeadline(context.Background(), f.q, id, now)
	if err != nil {
		f.t.Fatal(err)
	}
	return eventID
}

func TestHeadlineMatchingNeedsTwoKnownNames(t *testing.T) {
	f := newFixture(t)
	now := time.Now()
	protest := f.link(f.add(article{development: "Puma Shen responds to protesters at Jingmei Night Market", entities: []string{"沈伯洋", "景美夜市", "游智彬"}}))

	if got := f.match(f.headline("沈伯洋景美夜市掃街遇抗議 游智彬坦承是他"), now); got != protest.EventID {
		t.Fatalf("headline naming three of the event's entities: event %d, want %d", got, protest.EventID)
	}
	// One shared name is never enough: the same person is in many stories.
	if got := f.match(f.headline("沈伯洋公布交通政見 主打捷運路網"), now); got != 0 {
		t.Fatalf("headline sharing one name was attached to event %d", got)
	}
	if got := f.match(f.headline("中秋連假國道車流估增三成"), now); got != 0 {
		t.Fatalf("unrelated headline was attached to event %d", got)
	}
}

func TestHeadlineMatchingAddsNoStepAndDoesNotBumpTheEvent(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	res := f.link(f.add(article{development: "The council passes the budget", entities: []string{"範例市議會", "陳範例"}}))
	before, err := f.q.GetEvent(ctx, res.EventID)
	if err != nil {
		t.Fatal(err)
	}

	if got := f.match(f.headline("範例市議會三讀通過總預算 陳範例：感謝支持"), time.Now()); got != res.EventID {
		t.Fatalf("event %d, want %d", got, res.EventID)
	}
	steps, err := f.q.ListEventSteps(ctx, []int64{res.EventID})
	if err != nil || len(steps) != 1 {
		t.Fatalf("steps = %d (%v), want the one step the model wrote", len(steps), err)
	}
	after, err := f.q.GetEvent(ctx, res.EventID)
	if err != nil || !after.UpdatedAt.Equal(before.UpdatedAt) || after.ArticleCount != before.ArticleCount+1 {
		t.Fatalf("after = %+v, want one more article and the same updated time", after)
	}
}

func TestHeadlineMatchingIgnoresStaleEventsAndTies(t *testing.T) {
	f := newFixture(t)
	first := f.link(f.add(article{development: "Typhoon warning issued for Hualien", entities: []string{"花蓮縣", "中央氣象署"}}))

	// Long after the event went quiet, the same names mean a different story.
	if got := f.match(f.headline("中央氣象署：花蓮縣清晨有感地震"), time.Now().Add(30*24*time.Hour)); got != 0 {
		t.Fatalf("stale event %d was matched", got)
	}

	// Two current events fit equally well: don't guess.
	second := f.link(f.add(article{development: "Earthquake strikes off Hualien", entities: []string{"花蓮縣", "中央氣象署"}}))
	if second.EventID == first.EventID {
		t.Fatal("test setup: expected two separate events")
	}
	if got := f.match(f.headline("中央氣象署發布花蓮縣最新警報"), time.Now()); got != 0 {
		t.Fatalf("ambiguous headline was attached to event %d", got)
	}
}
