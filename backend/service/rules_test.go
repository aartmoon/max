package service

import "testing"

func TestClassification(t *testing.T) {
	for _, tc := range []struct{ text, want string }{
		{"ТРУБА течёт", "PIPE_LEAK"}, {"течет вода", "PIPE_LEAK"}, {"Протечка в подъезде", "PIPE_LEAK"},
		{"Сломан лифт", "ELEVATOR"}, {"Нет отопления, холодно", "HEATING"}, {"Батарея холодная", "HEATING"},
		{"Мусор во дворе", "OTHER"}, {"лифт и труба", "PIPE_LEAK"},
	} {
		if got := (RuleClassifier{}).Classify(tc.text); got != tc.want {
			t.Errorf("%q: got %s want %s", tc.text, got, tc.want)
		}
	}
}
func TestStatusTransitions(t *testing.T) {
	for _, tc := range []struct{ from, to string }{{"CREATED", "SENT"}, {"SENT", "ACCEPTED"}, {"ACCEPTED", "IN_PROGRESS"}, {"IN_PROGRESS", "RESOLVED"}} {
		got, err := NextStatus(tc.from)
		if err != nil || got != tc.to {
			t.Fatalf("%s: %s %v", tc.from, got, err)
		}
	}
	for _, terminal := range []string{"RESOLVED", "REJECTED", "INVALID"} {
		if _, err := NextStatus(terminal); err == nil {
			t.Errorf("expected rejection for %s", terminal)
		}
	}
}
