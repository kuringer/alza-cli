package client

import (
	"errors"
	"testing"
)

func TestClassifyOrderStatus(t *testing.T) {
	cases := map[string]string{
		"":                                PaymentStatusUnknown,
		"Počkajte prosím":                 PaymentStatusPending,
		"Čaká na platbu":                  PaymentStatusPending,
		"Nákup pripravujeme na odoslanie": PaymentStatusPaid,
		"Vybavená":                        PaymentStatusPaid,
		"Zrušená":                         PaymentStatusFailed,
	}
	for in, want := range cases {
		if got := classifyOrderStatus(in); got != want {
			t.Errorf("classifyOrderStatus(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPollOrderStatusWaitsUntilPaid(t *testing.T) {
	statuses := []string{"Počkajte prosím", "Počkajte prosím", "Nákup pripravujeme na odoslanie"}
	calls := 0
	fetch := func() ([]Order, error) {
		s := statuses[calls]
		calls++
		return []Order{{ID: "other", Status: "Vybavená"}, {ID: "42", Status: s}}, nil
	}
	payment, status := pollOrderStatus(fetch, "42", 6, 0)
	if payment != PaymentStatusPaid || status != "Nákup pripravujeme na odoslanie" || calls != 3 {
		t.Fatalf("got %q %q after %d calls", payment, status, calls)
	}
}

func TestPollOrderStatusStaysPendingAndSurvivesErrors(t *testing.T) {
	calls := 0
	fetch := func() ([]Order, error) {
		calls++
		if calls == 1 {
			return nil, errors.New("boom")
		}
		return []Order{{ID: "42", Status: "Počkajte prosím"}}, nil
	}
	payment, _ := pollOrderStatus(fetch, "42", 3, 0)
	if payment != PaymentStatusPending || calls != 3 {
		t.Fatalf("got %q after %d calls", payment, calls)
	}
}

func TestPollOrderStatusUnknownWhenOrderMissing(t *testing.T) {
	fetch := func() ([]Order, error) { return []Order{{ID: "1", Status: "Vybavená"}}, nil }
	if payment, _ := pollOrderStatus(fetch, "42", 2, 0); payment != PaymentStatusUnknown {
		t.Fatalf("got %q, want unknown", payment)
	}
}
