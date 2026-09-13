// Package constants 集中维护业务枚举、错误码、消息文案与日志模板。
package constants

// UserRole 用户角色枚举（JWT + RBAC 权限核心）
type UserRole string

const (
	RoleUser  UserRole = "user"  // 普通用户
	RoleAdmin UserRole = "admin" // 管理员
)

// GroupStatus 分账群组状态枚举
type GroupStatus string

const (
	GroupActive   GroupStatus = "active"   // 进行中
	GroupArchived GroupStatus = "archived" // 已归档
)

// ExpenseCategory 消费类别枚举
type ExpenseCategory string

const (
	CategoryDining    ExpenseCategory = "dining"    // 餐饮
	CategoryTransport ExpenseCategory = "transport" // 交通
	CategoryLodging   ExpenseCategory = "lodging"   // 住宿
	CategoryEntertain ExpenseCategory = "entertain" // 娱乐
	CategoryOther     ExpenseCategory = "other"     // 其他
)

// SplitType 分摊方式枚举
type SplitType string

const (
	SplitEqual  SplitType = "equal"  // 均摊
	SplitRatio  SplitType = "ratio"  // 按比例
	SplitAmount SplitType = "amount" // 按金额
)

// ExpenseStatus 消费记录状态枚举
type ExpenseStatus string

const (
	ExpenseActive   ExpenseStatus = "active"   // 有效
	ExpenseRefunded ExpenseStatus = "refunded" // 已退款
)

// SettlementStatus 结算建议状态枚举
type SettlementStatus string

const (
	SettlementPending SettlementStatus = "pending" // 待结算
	SettlementSettled SettlementStatus = "settled" // 已结算
)

// BudgetStatus 预算状态枚举（数据库存储值）
type BudgetStatus string

const (
	BudgetActive   BudgetStatus = "active"   // 生效中
	BudgetDisabled BudgetStatus = "disabled" // 已停用
)

// BudgetUsageStatus 预算执行状态枚举（按已用金额与预算金额实时推导，不落库）
type BudgetUsageStatus string

const (
	UsageNormal    BudgetUsageStatus = "normal"    // 正常（已用 < 预算）
	UsageExhausted BudgetUsageStatus = "exhausted" // 用尽（已用 = 预算）
	UsageExceeded  BudgetUsageStatus = "exceeded"  // 超出（已用 > 预算）
)

// BudgetRowStatus 月度预算执行统计行状态枚举（统计接口展示态）
type BudgetRowStatus string

const (
	BudgetRowUnset    BudgetRowStatus = "unset"    // 未设置预算
	BudgetRowActive   BudgetRowStatus = "active"   // 已设置且生效中
	BudgetRowDisabled BudgetRowStatus = "disabled" // 已设置但已停用
)

// AuditAction 审计动作枚举
type AuditAction string

const (
	ActionRegister           AuditAction = "register"
	ActionLogin              AuditAction = "login"
	ActionLogout             AuditAction = "logout"
	ActionGroupCreate        AuditAction = "group.create"
	ActionGroupUpdate        AuditAction = "group.update"
	ActionGroupArchive       AuditAction = "group.archive"
	ActionMemberInvite       AuditAction = "member.invite"
	ActionMemberRemove       AuditAction = "member.remove"
	ActionExpenseCreate      AuditAction = "expense.create"
	ActionExpenseUpdate      AuditAction = "expense.update"
	ActionExpenseDelete      AuditAction = "expense.delete"
	ActionExpenseExport      AuditAction = "expense.export"
	ActionSettlementGenerate AuditAction = "settlement.generate"
	ActionSettlementSettle   AuditAction = "settlement.settle"
	ActionBudgetCreate       AuditAction = "budget.create"
	ActionBudgetUpdate       AuditAction = "budget.update"
	ActionBudgetDisable      AuditAction = "budget.disable"
	ActionUserUpdate         AuditAction = "user.update"
	ActionUserRole           AuditAction = "user.role"
)

// IsValidUserRole 校验用户角色
func IsValidUserRole(r string) bool {
	return UserRole(r) == RoleUser || UserRole(r) == RoleAdmin
}

// IsValidGroupStatus 校验群组状态
func IsValidGroupStatus(s string) bool {
	return GroupStatus(s) == GroupActive || GroupStatus(s) == GroupArchived
}

// IsValidExpenseCategory 校验消费类别
func IsValidExpenseCategory(c string) bool {
	switch ExpenseCategory(c) {
	case CategoryDining, CategoryTransport, CategoryLodging, CategoryEntertain, CategoryOther:
		return true
	}
	return false
}

// IsValidSplitType 校验分摊方式
func IsValidSplitType(s string) bool {
	switch SplitType(s) {
	case SplitEqual, SplitRatio, SplitAmount:
		return true
	}
	return false
}

// IsValidExpenseStatus 校验消费状态
func IsValidExpenseStatus(s string) bool {
	return ExpenseStatus(s) == ExpenseActive || ExpenseStatus(s) == ExpenseRefunded
}

// IsValidSettlementStatus 校验结算状态
func IsValidSettlementStatus(s string) bool {
	return SettlementStatus(s) == SettlementPending || SettlementStatus(s) == SettlementSettled
}

// IsValidBudgetStatus 校验预算状态
func IsValidBudgetStatus(s string) bool {
	return BudgetStatus(s) == BudgetActive || BudgetStatus(s) == BudgetDisabled
}

// IsValidBudgetUsageStatus 校验预算执行状态
func IsValidBudgetUsageStatus(s string) bool {
	switch BudgetUsageStatus(s) {
	case UsageNormal, UsageExhausted, UsageExceeded:
		return true
	}
	return false
}

// IsValidBudgetRowStatus 校验预算执行统计行状态
func IsValidBudgetRowStatus(s string) bool {
	switch BudgetRowStatus(s) {
	case BudgetRowUnset, BudgetRowActive, BudgetRowDisabled:
		return true
	}
	return false
}
