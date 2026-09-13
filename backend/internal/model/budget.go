package model

import (
	"time"

	"github.com/aasplit/aasplit/internal/constants"
)

// Budget 群组月度类别预算实体：同一群组同一类别同一月份仅保留一条记录。
type Budget struct {
	ID        uint                    `gorm:"primaryKey" json:"id"`
	GroupID   uint                    `gorm:"uniqueIndex:idx_budget_group_cat_month;not null" json:"group_id"`
	Category  constants.ExpenseCategory `gorm:"size:32;uniqueIndex:idx_budget_group_cat_month;not null" json:"category"`
	Month     string                  `gorm:"size:7;uniqueIndex:idx_budget_group_cat_month;not null" json:"month"` // 格式 YYYY-MM
	Amount    float64                 `gorm:"type:double precision;not null" json:"amount"`
	Status    constants.BudgetStatus  `gorm:"size:16;not null;default:active" json:"status"`
	CreatedBy uint                    `json:"created_by"`
	CreatedAt time.Time               `json:"created_at"`
	UpdatedAt time.Time               `json:"updated_at"`
}

// TableName 指定表名。
func (Budget) TableName() string { return "budgets" }

// IsActive 判断预算是否生效中。
func (b *Budget) IsActive() bool { return b.Status == constants.BudgetActive }

// BudgetUsageStatusOf 按预算金额与已用金额推导执行状态（分位比较避免浮点误差）。
func BudgetUsageStatusOf(amount, used float64) constants.BudgetUsageStatus {
	amountCents := int64(amount*100 + 0.5)
	usedCents := int64(used*100 + 0.5)
	switch {
	case usedCents > amountCents:
		return constants.UsageExceeded
	case usedCents == amountCents:
		return constants.UsageExhausted
	default:
		return constants.UsageNormal
	}
}

// BudgetRatioOf 计算执行比例（已用 / 预算，保留四位小数；预算为 0 时比例为 0）。
func BudgetRatioOf(amount, used float64) float64 {
	if amount <= 0 {
		return 0
	}
	return float64(int64(used*10000/amount+0.5)) / 10000
}
