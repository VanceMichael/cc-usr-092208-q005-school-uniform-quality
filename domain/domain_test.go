package schooluniformquality

import (
    "encoding/json"
    "os"
    "strings"
    "testing"
)

func loadFixture(t *testing.T) Record {
    t.Helper()
    raw, err := os.ReadFile("../fixtures/domain.json")
    if err != nil { t.Fatal(err) }
    value, err := Parse(raw)
    if err != nil { t.Fatal(err) }
    return value
}

func reparse(t *testing.T, value Record) error {
    t.Helper()
    raw, err := json.Marshal(value)
    if err != nil { t.Fatal(err) }
    _, err = Parse(raw)
    return err
}

func TestFixtureMatchesDomain(t *testing.T) {
    value := loadFixture(t)
    if value.Domain != "school-uniform-quality" { t.Fatalf("领域标识不一致: %s", value.Domain) }
}

func TestComponentBalanceEnforced(t *testing.T) {
    value := loadFixture(t)
    // 让 S101 分发出去的上衣超过它持有的数量。
    for i := range value.Events {
        if value.Events[i].ID == "EV-07" {
            value.Events[i].Moves[0].Quantity = 500
        }
    }
    err := reparse(t, value)
    if err == nil || !strings.Contains(err.Error(), "不平衡") {
        t.Fatalf("应当发现部件数量不平衡, 实际: %v", err)
    }
}

func TestSplitSwapKeepsBalance(t *testing.T) {
    value := loadFixture(t)
    // 拆套换码若只记录一个方向且数量超过持有量，部件来源就断了。
    for i := range value.Events {
        if value.Events[i].Kind == "拆套换码" {
            value.Events[i].Moves = value.Events[i].Moves[:1]
            value.Events[i].Moves[0].Quantity = 500
        }
    }
    if err := reparse(t, value); err == nil {
        t.Fatal("拆套换码超发未被发现")
    }
}

func TestSettlementNeedsStandardAndReport(t *testing.T) {
    value := loadFixture(t)
    for i := range value.Events {
        if value.Events[i].Kind == "结算索赔" {
            value.Events[i].Evidence = []string{"LAB-P02"}
        }
    }
    if err := reparse(t, value); err == nil {
        t.Fatal("结算索赔缺少标准版本仍通过")
    }
}

func TestCorrectionNeedsExistingReport(t *testing.T) {
    value := loadFixture(t)
    for i := range value.Events {
        if value.Events[i].Kind == "实验室更正" {
            value.Events[i].Refs.Corrects = "LAB-NONE"
        }
    }
    if err := reparse(t, value); err == nil {
        t.Fatal("实验室更正指向不存在的报告仍通过")
    }
}

func TestFreezeNeedsRegisteredBatch(t *testing.T) {
    value := loadFixture(t)
    for i := range value.Events {
        if value.Events[i].Kind == "冻结" {
            value.Events[i].Refs.Batch = "BATCH-NONE"
        }
    }
    if err := reparse(t, value); err == nil {
        t.Fatal("冻结未登记批次仍通过")
    }
}

func TestAccessRulesCoverAllActors(t *testing.T) {
    value := loadFixture(t)
    value.AccessRules = value.AccessRules[:3]
    if err := reparse(t, value); err == nil {
        t.Fatal("缺少参与方权限约定仍通过")
    }
}

func TestRequiredOperationsPresent(t *testing.T) {
    value := loadFixture(t)
    kept := value.Events[:0]
    for _, ev := range value.Events {
        if ev.Kind != "学校间调剂" {
            kept = append(kept, ev)
        }
    }
    value.Events = kept
    if err := reparse(t, value); err == nil {
        t.Fatal("缺少学校间调剂事件仍通过")
    }
}

func TestUnknownEventKindRejected(t *testing.T) {
    value := loadFixture(t)
    value.Events[0].Kind = "私下处理"
    if err := reparse(t, value); err == nil {
        t.Fatal("未知事件类型仍通过")
    }
}
