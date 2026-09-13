package handler

import (
	"github.com/aasplit/aasplit/internal/dto"
	"github.com/aasplit/aasplit/internal/model"
	"github.com/aasplit/aasplit/internal/service"
	"github.com/aasplit/aasplit/internal/util"
	"github.com/gin-gonic/gin"
)

// BudgetHandler 群组月度预算 HTTP 处理器。
type BudgetHandler struct {
	budgetSvc *service.BudgetService
}

// NewBudgetHandler 构造预算处理器。
func NewBudgetHandler(budgetSvc *service.BudgetService) *BudgetHandler {
	return &BudgetHandler{budgetSvc: budgetSvc}
}

// Create 设置月度预算。
func (h *BudgetHandler) Create(c *gin.Context) {
	var req dto.CreateBudgetReq
	if err := c.ShouldBindJSON(&req); err != nil {
		util.Fail(c, util.Wrap(util.ValidationCode(err), util.ValidationMessage(err), err))
		return
	}
	if req.GroupID == 0 {
		req.GroupID = parseIDParam(c)
	}
	budget, err := h.budgetSvc.Create(util.GetUserID(c), req.GroupID, &req)
	if err != nil {
		util.Fail(c, err)
		return
	}
	util.Created(c, toBudgetResp(budget, 0))
}

// Update 调整预算金额。
func (h *BudgetHandler) Update(c *gin.Context) {
	var req dto.UpdateBudgetReq
	if err := c.ShouldBindJSON(&req); err != nil {
		util.Fail(c, util.Wrap(util.ValidationCode(err), util.ValidationMessage(err), err))
		return
	}
	if err := h.budgetSvc.Update(util.GetUserID(c), parseIDParam(c), &req); err != nil {
		util.Fail(c, err)
		return
	}
	util.OK(c, gin.H{"message": "预算调整成功"})
}

// Disable 停用预算。
func (h *BudgetHandler) Disable(c *gin.Context) {
	if err := h.budgetSvc.Disable(util.GetUserID(c), parseIDParam(c)); err != nil {
		util.Fail(c, err)
		return
	}
	util.OK(c, gin.H{"message": "预算已停用"})
}

// List 查询群组某月预算列表（含已用/剩余/执行比例）。
func (h *BudgetHandler) List(c *gin.Context) {
	groupID := parseIDParam(c)
	var query dto.BudgetQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		util.Fail(c, util.Wrap(util.ValidationCode(err), util.ValidationMessage(err), err))
		return
	}
	budgets, used, err := h.budgetSvc.List(util.GetUserID(c), groupID, query.Month)
	if err != nil {
		util.Fail(c, err)
		return
	}
	list := make([]*dto.BudgetResp, 0, len(budgets))
	for i := range budgets {
		list = append(list, toBudgetResp(&budgets[i], used[string(budgets[i].Category)]))
	}
	util.OK(c, list)
}

// Stats 按月统计各类别预算执行情况（未设置预算的类别保持未设置）。
func (h *BudgetHandler) Stats(c *gin.Context) {
	groupID := parseIDParam(c)
	var query dto.BudgetQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		util.Fail(c, util.Wrap(util.ValidationCode(err), util.ValidationMessage(err), err))
		return
	}
	stats, err := h.budgetSvc.Stats(util.GetUserID(c), groupID, query.Month)
	if err != nil {
		util.Fail(c, err)
		return
	}
	util.OK(c, stats)
}

// toBudgetResp 预算模型转响应（实时推导已用/剩余/执行比例/执行状态）。
func toBudgetResp(b *model.Budget, used float64) *dto.BudgetResp {
	used = util.Round2(used)
	return &dto.BudgetResp{
		ID:           b.ID,
		GroupID:      b.GroupID,
		Category:     string(b.Category),
		Month:        b.Month,
		Amount:       util.Round2(b.Amount),
		UsedAmount:   used,
		RemainAmount: util.Round2(b.Amount - used),
		Ratio:        model.BudgetRatioOf(b.Amount, used),
		UsageStatus:  string(model.BudgetUsageStatusOf(b.Amount, used)),
		Status:       string(b.Status),
		CreatedBy:    b.CreatedBy,
		CreatedAt:    util.FormatDateTime(b.CreatedAt),
		UpdatedAt:    util.FormatDateTime(b.UpdatedAt),
	}
}
