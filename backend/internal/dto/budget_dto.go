package dto

// CreateBudgetReq 设置月度预算请求（group_id 由路径参数补充）。
type CreateBudgetReq struct {
	GroupID  uint    `json:"group_id" binding:"omitempty,gt=0"`
	Category string  `json:"category" binding:"required,oneof=dining transport lodging entertain other"`
	Month    string  `json:"month" binding:"required"`
	Amount   float64 `json:"amount" binding:"required,gt=0"`
}

// UpdateBudgetReq 调整预算金额请求（金额必须为正数）。
type UpdateBudgetReq struct {
	Amount float64 `json:"amount" binding:"required,gt=0"`
}

// BudgetQuery 预算列表筛选查询。
type BudgetQuery struct {
	Month string `form:"month" binding:"omitempty"`
}

// BudgetResp 预算列表项响应（含实时推导的已用/剩余/执行比例/执行状态）。
type BudgetResp struct {
	ID           uint    `json:"id"`
	GroupID      uint    `json:"group_id"`
	Category     string  `json:"category"`
	Month        string  `json:"month"`
	Amount       float64 `json:"amount"`
	UsedAmount   float64 `json:"used_amount"`
	RemainAmount float64 `json:"remain_amount"`
	Ratio        float64 `json:"ratio"`
	UsageStatus  string  `json:"usage_status"`
	Status       string  `json:"status"`
	CreatedBy    uint    `json:"created_by"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

// BudgetExecution 月度预算执行统计行（未设置预算的类别保持 unset）。
type BudgetExecution struct {
	Category     string  `json:"category"`
	Status       string  `json:"status"` // unset / active / disabled
	BudgetID     uint    `json:"budget_id"`
	Amount       float64 `json:"amount"`
	UsedAmount   float64 `json:"used_amount"`
	RemainAmount float64 `json:"remain_amount"`
	Ratio        float64 `json:"ratio"`
	UsageStatus  string  `json:"usage_status"` // 生效中时：normal / exhausted / exceeded
}

// BudgetStatsResp 按月预算执行统计响应。
type BudgetStatsResp struct {
	Month       string            `json:"month"`
	Items       []BudgetExecution `json:"items"`
	TotalAmount float64           `json:"total_amount"`
	TotalUsed   float64           `json:"total_used"`
	TotalRemain float64           `json:"total_remain"`
	TotalRatio  float64           `json:"total_ratio"`
}
