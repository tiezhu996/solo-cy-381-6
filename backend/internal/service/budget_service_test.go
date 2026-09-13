package service

import (
	"testing"
	"time"

	"github.com/aasplit/aasplit/internal/constants"
	"github.com/aasplit/aasplit/internal/dto"
	"github.com/aasplit/aasplit/internal/model"
	"github.com/aasplit/aasplit/internal/repository"
	"github.com/aasplit/aasplit/internal/util"
	"gorm.io/gorm"
)

// newBudgetServiceFixture 构造带群组与成员的预算服务夹具（alice 群主，bob/carol 成员，dave 非成员）。
func newBudgetServiceFixture(t *testing.T) (*gorm.DB, *BudgetService, *ExpenseService, *GroupService, uint, uint, uint, uint, uint) {
	t.Helper()
	db := newTestDB(t)
	userRepo := repository.NewUserRepository(db)
	groupRepo := repository.NewGroupRepository(db)
	memberRepo := repository.NewGroupMemberRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	auditSvc := NewAuditService(auditRepo, newTestLogger())
	logger := newTestLogger()

	emails := map[string]string{"alice": "alice@budget.com", "bob": "bob@budget.com"}
	users := make(map[string]*model.User)
	for _, name := range []string{"alice", "bob", "carol", "dave"} {
		u := &model.User{Username: name, PasswordHash: "x", Nickname: name, Role: constants.RoleUser}
		if e, ok := emails[name]; ok {
			email := e
			u.Email = &email
		}
		if err := userRepo.Create(u); err != nil {
			t.Fatalf("create user %s: %v", name, err)
		}
		users[name] = u
	}
	groupSvc := NewGroupService(db, groupRepo, memberRepo, userRepo, auditSvc, logger)
	group, err := groupSvc.Create(users["alice"].ID, &dto.CreateGroupReq{Name: "预算测试群", Description: "测试"})
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	for _, name := range []string{"bob", "carol"} {
		if err := groupSvc.InviteMember(users["alice"].ID, group.ID, name); err != nil {
			t.Fatalf("invite %s: %v", name, err)
		}
	}
	budgetSvc := NewBudgetService(db, repository.NewBudgetRepository(db), memberRepo, groupRepo, auditSvc, logger)
	expenseSvc := NewExpenseService(db, repository.NewExpenseRepository(db), memberRepo, groupRepo, userRepo, auditSvc, logger)
	return db, budgetSvc, expenseSvc, groupSvc, group.ID, users["alice"].ID, users["bob"].ID, users["carol"].ID, users["dave"].ID
}

// currentMonthStr 当前月份 YYYY-MM。
func currentMonthStr() string { return time.Now().Format("2006-01") }

// nowPaidAt 当前时间字符串（消费记录 paid_at）。
func nowPaidAt() string { return time.Now().Format("2006-01-02 15:04:05") }

func TestBudgetServiceCreateValidation(t *testing.T) {
	_, svc, _, _, groupID, aliceID, _, _, daveID := newBudgetServiceFixture(t)
	month := currentMonthStr()
	future := time.Now().AddDate(0, 2, 0).Format("2006-01")

	tests := []struct {
		name    string
		userID  uint
		req     *dto.CreateBudgetReq
		wantErr bool
		errCode int
	}{
		{name: "valid current month", userID: aliceID, req: &dto.CreateBudgetReq{Category: "dining", Month: month, Amount: 300}},
		{name: "duplicate same category month", userID: aliceID, req: &dto.CreateBudgetReq{Category: "dining", Month: month, Amount: 500}, wantErr: true, errCode: constants.CodeBudgetExists},
		{name: "another category same month", userID: aliceID, req: &dto.CreateBudgetReq{Category: "transport", Month: month, Amount: 100}},
		{name: "future month allowed", userID: aliceID, req: &dto.CreateBudgetReq{Category: "dining", Month: future, Amount: 200}},
		{name: "past month rejected", userID: aliceID, req: &dto.CreateBudgetReq{Category: "lodging", Month: "2020-01", Amount: 100}, wantErr: true, errCode: constants.CodeBudgetPastMonth},
		{name: "invalid month format", userID: aliceID, req: &dto.CreateBudgetReq{Category: "lodging", Month: "2026-13", Amount: 100}, wantErr: true, errCode: constants.CodeValidationFailed},
		{name: "non-padded month rejected", userID: aliceID, req: &dto.CreateBudgetReq{Category: "lodging", Month: "2026-1", Amount: 100}, wantErr: true, errCode: constants.CodeValidationFailed},
		{name: "non-member rejected", userID: daveID, req: &dto.CreateBudgetReq{Category: "other", Month: month, Amount: 100}, wantErr: true, errCode: constants.CodeNotGroupMember},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Create(tt.userID, groupID, tt.req)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if ae := util.AsAppError(err); ae.Code != tt.errCode {
					t.Fatalf("err code = %d, want %d", ae.Code, tt.errCode)
				}
				return
			}
			if err != nil {
				t.Fatalf("create: %v", err)
			}
		})
	}
}

func TestBudgetServiceCreateArchivedGroup(t *testing.T) {
	_, svc, _, groupSvc, groupID, aliceID, _, _, _ := newBudgetServiceFixture(t)
	if err := groupSvc.Archive(aliceID, groupID); err != nil {
		t.Fatalf("archive: %v", err)
	}
	_, err := svc.Create(aliceID, groupID, &dto.CreateBudgetReq{Category: "dining", Month: currentMonthStr(), Amount: 100})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if ae := util.AsAppError(err); ae.Code != constants.CodeConflict {
		t.Fatalf("err code = %d, want %d", ae.Code, constants.CodeConflict)
	}
}

func TestBudgetServiceUpdateDisableReactivate(t *testing.T) {
	_, svc, _, _, groupID, aliceID, _, _, _ := newBudgetServiceFixture(t)
	month := currentMonthStr()

	budget, err := svc.Create(aliceID, groupID, &dto.CreateBudgetReq{Category: "dining", Month: month, Amount: 300})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := svc.Update(aliceID, budget.ID, &dto.UpdateBudgetReq{Amount: 450.5}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if err := svc.Disable(aliceID, budget.ID); err != nil {
		t.Fatalf("disable: %v", err)
	}
	// 已停用预算不可调整、不可重复停用
	if err := svc.Update(aliceID, budget.ID, &dto.UpdateBudgetReq{Amount: 600}); err == nil {
		t.Fatalf("expected disabled error on update, got nil")
	} else if ae := util.AsAppError(err); ae.Code != constants.CodeBudgetDisabled {
		t.Fatalf("update err code = %d, want %d", ae.Code, constants.CodeBudgetDisabled)
	}
	if err := svc.Disable(aliceID, budget.ID); err == nil {
		t.Fatalf("expected disabled error on second disable, got nil")
	}
	// 同类别同月份重新添加：复用原记录重新启用，仍只保留一条
	reactivated, err := svc.Create(aliceID, groupID, &dto.CreateBudgetReq{Category: "dining", Month: month, Amount: 800})
	if err != nil {
		t.Fatalf("re-create: %v", err)
	}
	if reactivated.ID != budget.ID {
		t.Fatalf("reactivated id = %d, want same record %d", reactivated.ID, budget.ID)
	}
	budgets, _, err := svc.List(aliceID, groupID, month)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(budgets) != 1 {
		t.Fatalf("budget count = %d, want 1", len(budgets))
	}
	if budgets[0].Amount != 800 || budgets[0].Status != constants.BudgetActive {
		t.Fatalf("budget = %+v, want amount 800 active", budgets[0])
	}
}

func TestBudgetServiceListSyncWithExpenses(t *testing.T) {
	_, svc, expenseSvc, _, groupID, aliceID, bobID, _ := newBudgetServiceFixture(t)
	month := currentMonthStr()

	if _, err := svc.Create(aliceID, groupID, &dto.CreateBudgetReq{Category: "dining", Month: month, Amount: 300}); err != nil {
		t.Fatalf("create budget: %v", err)
	}
	usedOf := func() (float64, string) {
		t.Helper()
		budgets, used, err := svc.List(aliceID, groupID, month)
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		if len(budgets) != 1 {
			t.Fatalf("budget count = %d, want 1", len(budgets))
		}
		b := budgets[0]
		u := util.Round2(used[string(b.Category)])
		return u, string(model.BudgetUsageStatusOf(b.Amount, u))
	}

	// 初始：无消费，已用 0，状态正常
	if used, status := usedOf(); used != 0 || status != string(constants.UsageNormal) {
		t.Fatalf("initial used = %.2f status = %s, want 0 normal", used, status)
	}

	createExpense := func(amount float64) *model.Expense {
		t.Helper()
		e, err := expenseSvc.Create(aliceID, &dto.CreateExpenseReq{
			GroupID: groupID, Title: "聚餐", Amount: amount, Category: "dining",
			PayerID: aliceID, SplitType: "equal", PaidAt: nowPaidAt(),
			Shares: []dto.ShareInput{{UserID: aliceID}, {UserID: bobID}},
		})
		if err != nil {
			t.Fatalf("create expense: %v", err)
		}
		return e
	}

	// 消费 120：已用同步为 120
	e1 := createExpense(120)
	if used, _ := usedOf(); used != 120 {
		t.Fatalf("used = %.2f, want 120", used)
	}
	// 再消费 180：累计 300，预算用尽
	e2 := createExpense(180)
	if used, status := usedOf(); used != 300 || status != string(constants.UsageExhausted) {
		t.Fatalf("used = %.2f status = %s, want 300 exhausted", used, status)
	}
	// 修改消费金额至 350：超出预算
	if err := expenseSvc.Update(aliceID, e2.ID, &dto.UpdateExpenseReq{
		Title: "聚餐", Amount: 350, Category: "dining", PayerID: aliceID,
		SplitType: "equal", PaidAt: nowPaidAt(),
		Shares: []dto.ShareInput{{UserID: aliceID}, {UserID: bobID}},
	}); err != nil {
		t.Fatalf("update expense: %v", err)
	}
	if used, status := usedOf(); used != 470 || status != string(constants.UsageExceeded) {
		t.Fatalf("used = %.2f status = %s, want 470 exceeded", used, status)
	}
	// 退款 350 的消费：已用恢复 120，状态回到正常
	if err := expenseSvc.Delete(aliceID, e2.ID); err != nil {
		t.Fatalf("refund expense: %v", err)
	}
	if used, status := usedOf(); used != 120 || status != string(constants.UsageNormal) {
		t.Fatalf("used = %.2f status = %s, want 120 normal", used, status)
	}
	// 作废（退款）剩余消费：已用归零
	if err := expenseSvc.Delete(aliceID, e1.ID); err != nil {
		t.Fatalf("refund expense: %v", err)
	}
	if used, status := usedOf(); used != 0 || status != string(constants.UsageNormal) {
		t.Fatalf("used = %.2f status = %s, want 0 normal", used, status)
	}
}

func TestBudgetServiceStats(t *testing.T) {
	_, svc, expenseSvc, _, groupID, aliceID, bobID, _ := newBudgetServiceFixture(t)
	month := currentMonthStr()

	if _, err := svc.Create(aliceID, groupID, &dto.CreateBudgetReq{Category: "dining", Month: month, Amount: 300}); err != nil {
		t.Fatalf("create dining budget: %v", err)
	}
	transport, err := svc.Create(aliceID, groupID, &dto.CreateBudgetReq{Category: "transport", Month: month, Amount: 100})
	if err != nil {
		t.Fatalf("create transport budget: %v", err)
	}
	if err := svc.Disable(aliceID, transport.ID); err != nil {
		t.Fatalf("disable transport: %v", err)
	}
	if _, err := expenseSvc.Create(aliceID, &dto.CreateExpenseReq{
		GroupID: groupID, Title: "大餐", Amount: 350, Category: "dining",
		PayerID: aliceID, SplitType: "equal", PaidAt: nowPaidAt(),
		Shares: []dto.ShareInput{{UserID: aliceID}, {UserID: bobID}},
	}); err != nil {
		t.Fatalf("create expense: %v", err)
	}

	stats, err := svc.Stats(aliceID, groupID, month)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if len(stats.Items) != 5 {
		t.Fatalf("stats items = %d, want 5", len(stats.Items))
	}
	byCat := make(map[string]dto.BudgetExecution, 5)
	for _, it := range stats.Items {
		byCat[it.Category] = it
	}
	dining := byCat["dining"]
	if dining.Status != string(constants.BudgetRowActive) || dining.UsedAmount != 350 || dining.UsageStatus != string(constants.UsageExceeded) {
		t.Fatalf("dining row = %+v, want active/350/exceeded", dining)
	}
	if dining.RemainAmount != -50 || dining.Ratio != 1.1667 {
		t.Fatalf("dining remain = %.2f ratio = %.4f, want -50 / 1.1667", dining.RemainAmount, dining.Ratio)
	}
	if tr := byCat["transport"]; tr.Status != string(constants.BudgetRowDisabled) || tr.UsageStatus != "" {
		t.Fatalf("transport row = %+v, want disabled", tr)
	}
	for _, cat := range []string{"lodging", "entertain", "other"} {
		if row := byCat[cat]; row.Status != string(constants.BudgetRowUnset) || row.Amount != 0 {
			t.Fatalf("%s row = %+v, want unset", cat, row)
		}
	}
	// 合计仅统计生效中的预算
	if stats.TotalAmount != 300 || stats.TotalUsed != 350 || stats.TotalRemain != -50 || stats.TotalRatio != 1.1667 {
		t.Fatalf("totals = %+v, want 300/350/-50/1.1667", stats)
	}
}

func TestBudgetServicePastMonthReadOnly(t *testing.T) {
	db, svc, _, _, groupID, aliceID, _, _, _ := newBudgetServiceFixture(t)
	past := time.Now().AddDate(0, -1, 0).Format("2006-01")

	// 直接落库一条已过月份预算（模拟历史数据）
	repo := repository.NewBudgetRepository(db)
	old := &model.Budget{GroupID: groupID, Category: constants.CategoryDining, Month: past, Amount: 200, Status: constants.BudgetActive, CreatedBy: aliceID}
	if err := repo.Create(nil, old); err != nil {
		t.Fatalf("seed past budget: %v", err)
	}
	// 已过月份可查看
	budgets, _, err := svc.List(aliceID, groupID, past)
	if err != nil || len(budgets) != 1 {
		t.Fatalf("list past month = %v len=%d, want 1 row", err, len(budgets))
	}
	if _, err := svc.Stats(aliceID, groupID, past); err != nil {
		t.Fatalf("stats past month: %v", err)
	}
	// 已过月份不可新增/调整/停用
	if _, err := svc.Create(aliceID, groupID, &dto.CreateBudgetReq{Category: "transport", Month: past, Amount: 100}); err == nil {
		t.Fatalf("expected past month create error, got nil")
	} else if ae := util.AsAppError(err); ae.Code != constants.CodeBudgetPastMonth {
		t.Fatalf("create err code = %d, want %d", ae.Code, constants.CodeBudgetPastMonth)
	}
	if err := svc.Update(aliceID, old.ID, &dto.UpdateBudgetReq{Amount: 300}); err == nil {
		t.Fatalf("expected past month update error, got nil")
	} else if ae := util.AsAppError(err); ae.Code != constants.CodeBudgetPastMonth {
		t.Fatalf("update err code = %d, want %d", ae.Code, constants.CodeBudgetPastMonth)
	}
	if err := svc.Disable(aliceID, old.ID); err == nil {
		t.Fatalf("expected past month disable error, got nil")
	} else if ae := util.AsAppError(err); ae.Code != constants.CodeBudgetPastMonth {
		t.Fatalf("disable err code = %d, want %d", ae.Code, constants.CodeBudgetPastMonth)
	}
}

func TestBudgetServiceUniqueConstraint(t *testing.T) {
	db, svc, _, _, groupID, aliceID, _, _, _ := newBudgetServiceFixture(t)
	month := currentMonthStr()
	if _, err := svc.Create(aliceID, groupID, &dto.CreateBudgetReq{Category: "dining", Month: month, Amount: 100}); err != nil {
		t.Fatalf("create: %v", err)
	}
	// 绕过 service 直接写库验证唯一索引兜底
	repo := repository.NewBudgetRepository(db)
	dup := &model.Budget{GroupID: groupID, Category: constants.CategoryDining, Month: month, Amount: 200, Status: constants.BudgetActive, CreatedBy: aliceID}
	if err := repo.Create(nil, dup); err == nil {
		t.Fatalf("expected unique constraint violation, got nil")
	}
}
