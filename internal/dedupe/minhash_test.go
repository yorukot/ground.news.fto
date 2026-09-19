package dedupe

import (
	"strings"
	"testing"
)

const wire = `（中央社記者範例台北十九日電）範例市政府今天將明年度總預算案送交市議會審議，歲出規模創歷年新高。` +
	`市府表示，增加的經費主要用於社會福利與公共運輸，包括長照據點擴充、公車路網調整與捷運延伸線的先期規劃。` +
	`市長陳範例受訪時表示，這份預算務實穩健，盼議會支持。多名議員則表示將嚴格審查舉債額度，` +
	`並要求市府說明各項新增計畫的必要性。議會預計下週開始分組審查，全案最快下月底完成三讀。`

func TestReprintsAreSimilar(t *testing.T) {
	// A typical reprint: different credit line, a trimmed sentence, other punctuation.
	reprint := strings.Replace(wire, "（中央社記者範例台北十九日電）", "〔記者範例／台北報導〕", 1)
	reprint = strings.Replace(reprint, "全案最快下月底完成三讀。", "", 1)
	reprint = strings.ReplaceAll(reprint, "，", ", ")

	got := Similarity(Sign(wire), Sign(reprint))
	if got < ReprintThreshold {
		t.Fatalf("reprint similarity = %.2f, want at least %.2f", got, ReprintThreshold)
	}
}

func TestDifferentArticlesOnTheSameEventAreNotReprints(t *testing.T) {
	other := `範例市明年度總預算案今天送進議會，規模再創新高，在野黨議員痛批市府財政紀律鬆散、債留子孫。` +
		`議員林範例在記者會上拿出圖表指出，市府舉債額度連續三年成長，卻拿不出具體的還款計畫，` +
		`質疑多項新增經費是為了選舉綁樁。市府發言人回應，債務仍在法定上限之內，社福支出不能等，` +
		`願意到議會逐項說明。地方人士觀察，這場預算攻防將是年底選戰的前哨戰。`

	got := Similarity(Sign(wire), Sign(other))
	if got >= ReprintThreshold {
		t.Fatalf("distinct articles similarity = %.2f, want below %.2f", got, ReprintThreshold)
	}
}

func TestIdenticalTextIsFullySimilar(t *testing.T) {
	if got := Similarity(Sign(wire), Sign(wire)); got != 1 {
		t.Fatalf("identical similarity = %.2f, want 1", got)
	}
}

func TestShortTextIsNotSigned(t *testing.T) {
	if sig := Sign("快訊：範例市長今天上午宣布參選。"); sig != nil {
		t.Fatal("very short text must not get a signature")
	}
	if got := Similarity(nil, Sign(wire)); got != 0 {
		t.Fatalf("similarity with an unsigned text = %.2f, want 0", got)
	}
}
