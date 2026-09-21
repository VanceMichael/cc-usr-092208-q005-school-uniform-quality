package schooluniformquality

import (
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

func TestFixtureMatchesDomain(t *testing.T) {
    value := loadFixture(t)
    if value.Domain != "school-uniform-quality" { t.Fatalf("领域标识不一致: %s", value.Domain) }
}

func TestFixtureCoversEvidenceChain(t *testing.T) {
    value := loadFixture(t)
    want := []string{"供应合同", "款号", "面辅料", "生产批次", "抽样封签", "检验项目", "到校验收", "分发去向", "退换处理"}
    if len(value.EvidenceChain) != len(want) { t.Fatalf("证据链环节数量不一致: %d", len(value.EvidenceChain)) }
    for i, stage := range want {
        if value.EvidenceChain[i] != stage { t.Fatalf("证据链第 %d 环不一致: %s", i+1, value.EvidenceChain[i]) }
    }
}

func TestFixtureCoversBalanceOperations(t *testing.T) {
    value := loadFixture(t)
    for _, operation := range []string{"拆套换码", "补货", "复检", "实验室更正", "学校间调剂", "学生退回"} {
        found := false
        for _, item := range value.Operations {
            if item == operation { found = true }
        }
        if !found { t.Fatalf("缺少保持部件来源与数量平衡的操作: %s", operation) }
    }
}

func TestFixtureStatesAccessBoundaries(t *testing.T) {
    value := loadFixture(t)
    if len(value.AccessRules) != 3 { t.Fatalf("访问边界数量不一致: %d", len(value.AccessRules)) }
    joined := strings.Join(value.AccessRules, ";")
    for _, role := range []string{"厂家", "学校", "监管"} {
        if !strings.Contains(joined, role) { t.Fatalf("访问边界缺少角色: %s", role) }
    }
}
