package model

import (
	"testing"

	"github.com/aasplit/aasplit/internal/constants"
)

func TestBudgetUsageStatusOf(t *testing.T) {
	tests := []struct {
		name   string
		amount float64
		used   float64
		want   constants.BudgetUsageStatus
	}{
		{name: "zero used", amount: 300, used: 0, want: constants.UsageNormal},
		{name: "below budget", amount: 300, used: 299.99, want: constants.UsageNormal},
		{name: "exactly exhausted", amount: 300, used: 300, want: constants.UsageExhausted},
		{name: "exhausted with cents", amount: 123.45, used: 123.45, want: constants.UsageExhausted},
		{name: "one cent over", amount: 300, used: 300.01, want: constants.UsageExceeded},
		{name: "far exceeded", amount: 100, used: 500, want: constants.UsageExceeded},
		{name: "float noise tolerated", amount: 0.3, used: 0.1 + 0.2, want: constants.UsageExhausted},
		{name: "one cent below", amount: 0.3, used: 0.29, want: constants.UsageNormal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BudgetUsageStatusOf(tt.amount, tt.used); got != tt.want {
				t.Fatalf("BudgetUsageStatusOf(%.2f, %.4f) = %s, want %s", tt.amount, tt.used, got, tt.want)
			}
		})
	}
}

func TestBudgetRatioOf(t *testing.T) {
	tests := []struct {
		name   string
		amount float64
		used   float64
		want   float64
	}{
		{name: "zero budget", amount: 0, used: 100, want: 0},
		{name: "half", amount: 300, used: 150, want: 0.5},
		{name: "full", amount: 300, used: 300, want: 1},
		{name: "over", amount: 300, used: 350, want: 1.1667},
		{name: "tiny", amount: 10000, used: 1, want: 0.0001},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BudgetRatioOf(tt.amount, tt.used); got != tt.want {
				t.Fatalf("BudgetRatioOf(%.2f, %.2f) = %.4f, want %.4f", tt.amount, tt.used, got, tt.want)
			}
		})
	}
}
