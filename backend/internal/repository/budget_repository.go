package repository

import (
	"errors"
	"fmt"
	"time"

	"github.com/aasplit/aasplit/internal/model"
	"gorm.io/gorm"
)

// ErrBudgetNotFound 预算不存在哨兵错误。
var ErrBudgetNotFound = errors.New("budget not found")

// BudgetRepository 群组月度预算数据访问。
type BudgetRepository struct {
	db *gorm.DB
}

// NewBudgetRepository 构造预算仓储。
func NewBudgetRepository(db *gorm.DB) *BudgetRepository {
	return &BudgetRepository{db: db}
}

// Create 创建预算记录（事务中调用）。
func (r *BudgetRepository) Create(tx *gorm.DB, budget *model.Budget) error {
	if tx == nil {
		tx = r.db
	}
	if err := tx.Create(budget).Error; err != nil {
		return fmt.Errorf("create budget: %w", err)
	}
	return nil
}

// FindByID 按 ID 查询预算。
func (r *BudgetRepository) FindByID(id uint) (*model.Budget, error) {
	var b model.Budget
	if err := r.db.First(&b, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBudgetNotFound
		}
		return nil, fmt.Errorf("find budget by id: %w", err)
	}
	return &b, nil
}

// FindByGroupCategoryMonth 按 群组+类别+月份 查询预算（唯一约束键）。
func (r *BudgetRepository) FindByGroupCategoryMonth(groupID uint, category, month string) (*model.Budget, error) {
	var b model.Budget
	if err := r.db.Where("group_id = ? AND category = ? AND month = ?", groupID, category, month).First(&b).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBudgetNotFound
		}
		return nil, fmt.Errorf("find budget by group/category/month: %w", err)
	}
	return &b, nil
}

// Update 更新预算记录（事务中调用）。
func (r *BudgetRepository) Update(tx *gorm.DB, budget *model.Budget) error {
	if tx == nil {
		tx = r.db
	}
	if err := tx.Save(budget).Error; err != nil {
		return fmt.Errorf("update budget: %w", err)
	}
	return nil
}

// ListByGroupMonth 查询群组某月的全部预算（含已停用，按类别排序）。
func (r *BudgetRepository) ListByGroupMonth(groupID uint, month string) ([]model.Budget, error) {
	var budgets []model.Budget
	if err := r.db.Where("group_id = ? AND month = ?", groupID, month).
		Order("category ASC").Find(&budgets).Error; err != nil {
		return nil, fmt.Errorf("list budgets by group/month: %w", err)
	}
	return budgets, nil
}

// SumUsedByMonth 汇总群组某月各消费类别的有效消费金额（已用金额；复用：预算列表与月度执行统计共用）。
func (r *BudgetRepository) SumUsedByMonth(groupID uint, start, end time.Time) (map[string]float64, error) {
	type row struct {
		Category string
		Amount   float64
	}
	var rows []row
	if err := r.db.Model(&model.Expense{}).
		Select("category, COALESCE(SUM(amount),0) AS amount").
		Where("group_id = ? AND status = ? AND paid_at >= ? AND paid_at < ?", groupID, "active", start, end).
		Group("category").Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("sum used by month: %w", err)
	}
	used := make(map[string]float64, len(rows))
	for _, rr := range rows {
		used[rr.Category] = rr.Amount
	}
	return used, nil
}
