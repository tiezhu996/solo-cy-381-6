package service

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/aasplit/aasplit/internal/constants"
	"github.com/aasplit/aasplit/internal/dto"
	"github.com/aasplit/aasplit/internal/model"
	"github.com/aasplit/aasplit/internal/repository"
	"github.com/aasplit/aasplit/internal/util"
	"gorm.io/gorm"
)

// BudgetService 群组月度预算业务逻辑。
type BudgetService struct {
	db         *gorm.DB
	budgetRepo *repository.BudgetRepository
	memberRepo *repository.GroupMemberRepository
	groupRepo  *repository.GroupRepository
	auditSvc   *AuditService
	logger     *slog.Logger
}

// NewBudgetService 构造预算服务。
func NewBudgetService(db *gorm.DB, budgetRepo *repository.BudgetRepository, memberRepo *repository.GroupMemberRepository, groupRepo *repository.GroupRepository, auditSvc *AuditService, logger *slog.Logger) *BudgetService {
	return &BudgetService{db: db, budgetRepo: budgetRepo, memberRepo: memberRepo, groupRepo: groupRepo, auditSvc: auditSvc, logger: logger}
}

// budgetCategories 预算支持的消费类别（与 ExpenseCategory 枚举保持一致顺序）。
var budgetCategories = []constants.ExpenseCategory{
	constants.CategoryDining,
	constants.CategoryTransport,
	constants.CategoryLodging,
	constants.CategoryEntertain,
	constants.CategoryOther,
}

// Create 设置月度预算（事务 + 群组行锁；同类别同月份已停用的记录将被重新启用）。
func (s *BudgetService) Create(userID, groupID uint, req *dto.CreateBudgetReq) (*model.Budget, error) {
	if !isValidBudgetMonth(req.Month) {
		return nil, util.NewAppError(constants.CodeValidationFailed, constants.MsgErrBudgetMonth, nil)
	}
	if isPastBudgetMonth(req.Month) {
		return nil, util.NewAppError(constants.CodeBudgetPastMonth, fmt.Sprintf("预算 budget 月份 month=%s 已过，%s", req.Month, constants.MsgErrBudgetPastMonth), nil)
	}
	var created *model.Budget
	err := s.db.Transaction(func(tx *gorm.DB) error {
		group, err := s.groupRepo.LockByID(groupID)
		if err != nil {
			return util.Wrap(constants.CodeNotFound, "群组 group 不存在", err)
		}
		if !group.IsActive() {
			return util.NewAppError(constants.CodeConflict, "群组 group 已归档，无法设置预算 budget", nil)
		}
		if err := s.ensureMember(tx, groupID, userID); err != nil {
			return err
		}
		existing, err := s.budgetRepo.FindByGroupCategoryMonth(groupID, req.Category, req.Month)
		if err != nil && !errors.Is(err, repository.ErrBudgetNotFound) {
			return util.Wrap(constants.CodeInternalError, constants.MsgErrInternal, err)
		}
		if err == nil {
			if existing.IsActive() {
				return util.NewAppError(constants.CodeBudgetExists, fmt.Sprintf("类别 category=%s 在月份 month=%s 的预算 budget 已存在，%s", req.Category, req.Month, constants.MsgErrBudgetExists), nil)
			}
			// 同类别同月份仅保留一条：已停用记录直接重新启用并更新金额。
			existing.Amount = util.Round2(req.Amount)
			existing.Status = constants.BudgetActive
			if err := s.budgetRepo.Update(tx, existing); err != nil {
				return util.Wrap(constants.CodeInternalError, constants.MsgErrInternal, err)
			}
			created = existing
			return nil
		}
		budget := &model.Budget{
			GroupID:   groupID,
			Category:  constants.ExpenseCategory(req.Category),
			Month:     req.Month,
			Amount:    util.Round2(req.Amount),
			Status:    constants.BudgetActive,
			CreatedBy: userID,
		}
		if err := s.budgetRepo.Create(tx, budget); err != nil {
			return util.Wrap(constants.CodeInternalError, constants.MsgErrInternal, err)
		}
		created = budget
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.logger.Info(fmt.Sprintf(constants.LogBudgetCreated, created.ID, created.GroupID, created.Category, created.Month, created.Amount, userID))
	s.auditSvc.Record(userID, string(constants.ActionBudgetCreate), "budget", fmt.Sprint(created.ID), fmt.Sprintf("设置预算 %s %s 金额 %.2f", created.Month, util.CategoryText(string(created.Category)), created.Amount), "")
	return created, nil
}

// Update 调整预算金额（事务 + 群组行锁；仅生效中的预算可调整）。
func (s *BudgetService) Update(userID, budgetID uint, req *dto.UpdateBudgetReq) error {
	var budget *model.Budget
	err := s.db.Transaction(func(tx *gorm.DB) error {
		b, err := s.writableBudget(tx, userID, budgetID)
		if err != nil {
			return err
		}
		if !b.IsActive() {
			return util.NewAppError(constants.CodeBudgetDisabled, fmt.Sprintf("预算 budget_id=%d 已停用，%s", budgetID, constants.MsgErrBudgetDisabled), nil)
		}
		b.Amount = util.Round2(req.Amount)
		if err := s.budgetRepo.Update(tx, b); err != nil {
			return util.Wrap(constants.CodeInternalError, constants.MsgErrInternal, err)
		}
		budget = b
		return nil
	})
	if err != nil {
		return err
	}
	s.logger.Info(fmt.Sprintf(constants.LogBudgetUpdated, budget.ID, budget.GroupID, budget.Category, budget.Month, budget.Amount, userID))
	s.auditSvc.Record(userID, string(constants.ActionBudgetUpdate), "budget", fmt.Sprint(budget.ID), fmt.Sprintf("调整预算 %s %s 金额为 %.2f", budget.Month, util.CategoryText(string(budget.Category)), budget.Amount), "")
	return nil
}

// Disable 停用预算（事务 + 群组行锁；保留记录，同类别同月份仍只保留一条）。
func (s *BudgetService) Disable(userID, budgetID uint) error {
	var budget *model.Budget
	err := s.db.Transaction(func(tx *gorm.DB) error {
		b, err := s.writableBudget(tx, userID, budgetID)
		if err != nil {
			return err
		}
		if !b.IsActive() {
			return util.NewAppError(constants.CodeBudgetDisabled, fmt.Sprintf("预算 budget_id=%d 已停用，无需重复操作", budgetID), nil)
		}
		b.Status = constants.BudgetDisabled
		if err := s.budgetRepo.Update(tx, b); err != nil {
			return util.Wrap(constants.CodeInternalError, constants.MsgErrInternal, err)
		}
		budget = b
		return nil
	})
	if err != nil {
		return err
	}
	s.logger.Info(fmt.Sprintf(constants.LogBudgetDisabled, budget.ID, budget.GroupID, budget.Category, budget.Month, userID))
	s.auditSvc.Record(userID, string(constants.ActionBudgetDisable), "budget", fmt.Sprint(budget.ID), fmt.Sprintf("停用预算 %s %s", budget.Month, util.CategoryText(string(budget.Category))), "")
	return nil
}

// writableBudget 校验预算可写（事务内调用）：存在、群组进行中、操作者为成员、月份未过期。
func (s *BudgetService) writableBudget(tx *gorm.DB, userID, budgetID uint) (*model.Budget, error) {
	b, err := s.budgetRepo.FindByID(budgetID)
	if errors.Is(err, repository.ErrBudgetNotFound) {
		return nil, util.NewAppError(constants.CodeNotFound, "预算 budget 不存在", err)
	}
	if err != nil {
		return nil, util.Wrap(constants.CodeInternalError, constants.MsgErrInternal, err)
	}
	group, err := s.groupRepo.LockByID(b.GroupID)
	if err != nil {
		return nil, util.Wrap(constants.CodeNotFound, "群组 group 不存在", err)
	}
	if !group.IsActive() {
		return nil, util.NewAppError(constants.CodeConflict, "群组 group 已归档，无法修改预算 budget", nil)
	}
	if err := s.ensureMember(tx, b.GroupID, userID); err != nil {
		return nil, err
	}
	if isPastBudgetMonth(b.Month) {
		return nil, util.NewAppError(constants.CodeBudgetPastMonth, fmt.Sprintf("预算 budget 月份 month=%s 已过，%s", b.Month, constants.MsgErrBudgetPastMonth), nil)
	}
	return b, nil
}

// List 查询群组某月预算列表（含已用金额、剩余金额、执行比例与执行状态）。
func (s *BudgetService) List(userID, groupID uint, month string) ([]model.Budget, map[string]float64, error) {
	if month == "" {
		month = currentBudgetMonth()
	}
	if !isValidBudgetMonth(month) {
		return nil, nil, util.NewAppError(constants.CodeValidationFailed, constants.MsgErrBudgetMonth, nil)
	}
	if err := s.ensureGroupMember(groupID, userID); err != nil {
		return nil, nil, err
	}
	budgets, err := s.budgetRepo.ListByGroupMonth(groupID, month)
	if err != nil {
		return nil, nil, util.Wrap(constants.CodeInternalError, constants.MsgErrInternal, err)
	}
	used, err := s.sumUsed(groupID, month)
	if err != nil {
		return nil, nil, err
	}
	return budgets, used, nil
}

// Stats 按月统计全部消费类别的预算执行情况（未设置预算的类别保持未设置）。
func (s *BudgetService) Stats(userID, groupID uint, month string) (*dto.BudgetStatsResp, error) {
	if month == "" {
		month = currentBudgetMonth()
	}
	if !isValidBudgetMonth(month) {
		return nil, util.NewAppError(constants.CodeValidationFailed, constants.MsgErrBudgetMonth, nil)
	}
	if err := s.ensureGroupMember(groupID, userID); err != nil {
		return nil, err
	}
	budgets, err := s.budgetRepo.ListByGroupMonth(groupID, month)
	if err != nil {
		return nil, util.Wrap(constants.CodeInternalError, constants.MsgErrInternal, err)
	}
	used, err := s.sumUsed(groupID, month)
	if err != nil {
		return nil, err
	}
	byCategory := make(map[string]*model.Budget, len(budgets))
	for i := range budgets {
		byCategory[string(budgets[i].Category)] = &budgets[i]
	}
	resp := &dto.BudgetStatsResp{Month: month, Items: make([]dto.BudgetExecution, 0, len(budgetCategories))}
	for _, cat := range budgetCategories {
		category := string(cat)
		usedAmount := util.Round2(used[category])
		b, ok := byCategory[category]
		if !ok {
			// 未设置预算的类别保持未设置。
			resp.Items = append(resp.Items, dto.BudgetExecution{Category: category, Status: string(constants.BudgetRowUnset), UsedAmount: usedAmount})
			continue
		}
		row := dto.BudgetExecution{
			Category:   category,
			BudgetID:   b.ID,
			Amount:     util.Round2(b.Amount),
			UsedAmount: usedAmount,
		}
		if !b.IsActive() {
			row.Status = string(constants.BudgetRowDisabled)
			resp.Items = append(resp.Items, row)
			continue
		}
		row.Status = string(constants.BudgetRowActive)
		row.RemainAmount = util.Round2(b.Amount - usedAmount)
		row.Ratio = model.BudgetRatioOf(b.Amount, usedAmount)
		row.UsageStatus = string(model.BudgetUsageStatusOf(b.Amount, usedAmount))
		resp.TotalAmount += b.Amount
		resp.TotalUsed += usedAmount
		resp.Items = append(resp.Items, row)
	}
	resp.TotalAmount = util.Round2(resp.TotalAmount)
	resp.TotalUsed = util.Round2(resp.TotalUsed)
	resp.TotalRemain = util.Round2(resp.TotalAmount - resp.TotalUsed)
	resp.TotalRatio = model.BudgetRatioOf(resp.TotalAmount, resp.TotalUsed)
	return resp, nil
}

// sumUsed 汇总群组某月各类别已用金额（有效消费记录实时求和，消费创建/修改/退款/作废后自动同步）。
func (s *BudgetService) sumUsed(groupID uint, month string) (map[string]float64, error) {
	start, end := budgetMonthRange(month)
	used, err := s.budgetRepo.SumUsedByMonth(groupID, start, end)
	if err != nil {
		return nil, util.Wrap(constants.CodeInternalError, constants.MsgErrInternal, err)
	}
	return used, nil
}

// ensureGroupMember 校验用户是群组成员（读操作前置）。
func (s *BudgetService) ensureGroupMember(groupID, userID uint) error {
	ok, err := s.memberRepo.Exists(groupID, userID)
	if err != nil {
		return util.Wrap(constants.CodeInternalError, constants.MsgErrInternal, err)
	}
	if !ok {
		return util.NewAppError(constants.CodeNotGroupMember, constants.MsgErrNotGroupMember, nil)
	}
	return nil
}

// ensureMember 校验用户是群组成员（事务内调用）。
func (s *BudgetService) ensureMember(tx *gorm.DB, groupID, userID uint) error {
	var n int64
	if err := tx.Model(&model.GroupMember{}).Where("group_id = ? AND user_id = ? AND status = ?", groupID, userID, "active").Count(&n).Error; err != nil {
		return util.Wrap(constants.CodeInternalError, constants.MsgErrInternal, err)
	}
	if n == 0 {
		return util.NewAppError(constants.CodeNotGroupMember, fmt.Sprintf("用户 user_id=%d 不是群组 group_id=%d 的成员 member", userID, groupID), nil)
	}
	return nil
}

// currentBudgetMonth 当前月份（YYYY-MM，本地时区）。
func currentBudgetMonth() string {
	return time.Now().Format("2006-01")
}

// isValidBudgetMonth 校验月份格式为 YYYY-MM。
func isValidBudgetMonth(month string) bool {
	t, err := time.ParseInLocation("2006-01", month, time.Local)
	return err == nil && t.Format("2006-01") == month
}

// isPastBudgetMonth 判断月份是否已过（早于当前月份；已过月份仅可查看）。
func isPastBudgetMonth(month string) bool {
	return month < currentBudgetMonth()
}

// budgetMonthRange 返回月份对应的本地时间区间 [start, end)。
func budgetMonthRange(month string) (time.Time, time.Time) {
	t, _ := time.ParseInLocation("2006-01", month, time.Local)
	start := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.Local)
	return start, start.AddDate(0, 1, 0)
}
