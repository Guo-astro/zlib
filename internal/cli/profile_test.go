package cli

import (
	"strings"
	"testing"

	zlib "github.com/heartleo/zlib"
)

func TestRenderProfileCardShowsAccountAndKnownQuota(t *testing.T) {
	premium := true
	profile := zlib.UserProfile{
		Name: "Reader", Email: "reader@example.com", Premium: &premium,
		QuotaKnown: true,
		DownloadLimit: zlib.DownloadLimit{
			DailyAmount: 3, DailyAllowed: 10, DailyRemaining: 7,
		},
	}
	card := renderProfileCard(profile)
	for _, want := range []string{"Reader", "reader@example.com", "Premium", "3 / 10 used"} {
		if !strings.Contains(card, want) {
			t.Errorf("profile card is missing %q", want)
		}
	}
	if strings.Contains(card, "Remaining") {
		t.Fatal("profile repeats the quota as Remaining")
	}
}

func TestRenderProfileCardDoesNotClaimUnknownQuotaIsZero(t *testing.T) {
	card := renderProfileCard(zlib.UserProfile{Name: "Reader"})
	if !strings.Contains(card, "Quota unavailable") || strings.Contains(card, "0 / 0") {
		t.Fatalf("profile card = %q", card)
	}
}
