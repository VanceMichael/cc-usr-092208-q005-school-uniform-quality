package schooluniformquality

import (
    "encoding/json"
    "errors"
    "fmt"
)

// Record 表示项目共享的领域资料。
type Record struct {
    Domain      string       `json:"domain"`
    Version     int          `json:"version"`
    SampleID    string       `json:"sample_id"`
    Actors      []string     `json:"actors"`
    Facts       []string     `json:"facts"`
    Constraints []string     `json:"constraints"`
    Components  []string     `json:"components"`
    ChainStages []string     `json:"chain_stages"`
    Standards   []Standard   `json:"standards"`
    AccessRules []AccessRule `json:"access_rules"`
    Events      []ChainEvent `json:"events"`
}

// Standard 表示证据链中采用的标准版本。
type Standard struct {
    Code    string `json:"code"`
    Version string `json:"version"`
    Title   string `json:"title"`
}

// Ref 返回结算与检验中引用标准的写法。
func (s Standard) Ref() string { return s.Code + "-" + s.Version }

// AccessRule 表示一个参与方允许看到和操作的资料范围。
type AccessRule struct {
    Actor       string   `json:"actor"`
    Scope       string   `json:"scope"`
    Permissions []string `json:"permissions"`
}

// Move 表示一个部件在持有方之间的数量流转。
type Move struct {
    Component string `json:"component"`
    From      string `json:"from"`
    To        string `json:"to"`
    Quantity  int    `json:"quantity"`
    Size      string `json:"size,omitempty"`
}

// EventRefs 表示事件挂接的证据链引用。
type EventRefs struct {
    Contract      string   `json:"contract,omitempty"`
    StyleNo       string   `json:"style_no,omitempty"`
    Batch         string   `json:"batch,omitempty"`
    Report        string   `json:"report,omitempty"`
    Corrects      string   `json:"corrects,omitempty"`
    Standard      string   `json:"standard,omitempty"`
    Seals         []string `json:"seals,omitempty"`
    Materials     []string `json:"materials,omitempty"`
    NotifyTargets []string `json:"notify_targets,omitempty"`
}

// ChainEvent 表示证据链上的一个事件。
type ChainEvent struct {
    ID       string    `json:"id"`
    Kind     string    `json:"kind"`
    Note     string    `json:"note"`
    Refs     EventRefs `json:"refs,omitempty"`
    Moves    []Move    `json:"moves,omitempty"`
    Evidence []string  `json:"evidence,omitempty"`
}

// 证据链环节的固定顺序。
var canonicalStages = []string{
    "供应合同", "款号", "面辅料", "生产批次", "抽样封签",
    "检验项目", "到校验收", "分发去向", "退换处理",
}

// 入库类事件只增加持有量，出库来源记为生产侧。
var additiveKinds = map[string]bool{"批次登记": true, "补货": true}

// 流转类事件在持有方之间移动部件，总量守恒。
var conservingKinds = map[string]bool{
    "到校验收": true, "分发": true, "拆套换码": true,
    "学校间调剂": true, "学生退回": true,
}

// 凭证类事件不改变部件数量。
var evidenceKinds = map[string]bool{
    "合同登记": true, "抽样封签": true, "检验": true, "复检": true,
    "实验室更正": true, "冻结": true, "通知": true, "结算索赔": true,
}

// Parse 读取并检查带版本的业务资料。
func Parse(raw []byte) (Record, error) {
    var value Record
    if err := json.Unmarshal(raw, &value); err != nil {
        return Record{}, err
    }
    if err := value.Validate(); err != nil {
        return Record{}, err
    }
    return value, nil
}

// Validate 检查领域资料的完整性与证据链不变量。
func (value Record) Validate() error {
    if value.Domain == "" || value.Version < 2 || value.SampleID == "" ||
        len(value.Actors) < 2 || len(value.Facts) < 2 || len(value.Constraints) < 2 {
        return errors.New("共享资料缺少必要字段")
    }
    if err := value.validateStages(); err != nil {
        return err
    }
    if err := value.validateAccess(); err != nil {
        return err
    }
    return value.validateEvents()
}

func (value Record) validateStages() error {
    if len(value.Components) < 2 {
        return errors.New("至少登记上衣与裤子两个部件")
    }
    seen := map[string]bool{}
    for _, c := range value.Components {
        if c == "" || seen[c] {
            return fmt.Errorf("部件名称重复或为空: %q", c)
        }
        seen[c] = true
    }
    if len(value.ChainStages) != len(canonicalStages) {
        return errors.New("证据链环节不完整")
    }
    for i, stage := range canonicalStages {
        if value.ChainStages[i] != stage {
            return fmt.Errorf("证据链环节顺序不符: 第%d环应为%s", i+1, stage)
        }
    }
    if len(value.Standards) == 0 {
        return errors.New("缺少采用的标准版本")
    }
    for _, s := range value.Standards {
        if s.Code == "" || s.Version == "" || s.Title == "" {
            return fmt.Errorf("标准条目缺少编号、版本或名称: %+v", s)
        }
    }
    return nil
}

func (value Record) validateAccess() error {
    covered := map[string]bool{}
    for _, rule := range value.AccessRules {
        if rule.Actor == "" || rule.Scope == "" || len(rule.Permissions) == 0 {
            return fmt.Errorf("权限条目不完整: %+v", rule)
        }
        if covered[rule.Actor] {
            return fmt.Errorf("参与方权限重复登记: %s", rule.Actor)
        }
        covered[rule.Actor] = true
    }
    for _, actor := range value.Actors {
        if !covered[actor] {
            return fmt.Errorf("参与方缺少权限约定: %s", actor)
        }
    }
    return nil
}

func (value Record) validateEvents() error {
    if len(value.Events) == 0 {
        return errors.New("证据链缺少事件")
    }
    componentOf := map[string]bool{}
    for _, c := range value.Components {
        componentOf[c] = true
    }
    standardRefs := map[string]bool{}
    for _, s := range value.Standards {
        standardRefs[s.Ref()] = true
    }
    ids := map[string]bool{}
    batches := map[string]bool{}
    reports := map[string]bool{}
    kindsSeen := map[string]bool{}
    // balance[部件][持有方] = 当前数量
    balance := map[string]map[string]int{}
    for _, ev := range value.Events {
        if ev.ID == "" || ev.Note == "" {
            return fmt.Errorf("事件缺少编号或说明: %+v", ev)
        }
        if ids[ev.ID] {
            return fmt.Errorf("事件编号重复: %s", ev.ID)
        }
        ids[ev.ID] = true
        switch {
        case additiveKinds[ev.Kind], conservingKinds[ev.Kind]:
            if len(ev.Moves) == 0 {
                return fmt.Errorf("事件%s(%s)缺少部件流转", ev.ID, ev.Kind)
            }
        case evidenceKinds[ev.Kind]:
            if len(ev.Moves) > 0 {
                return fmt.Errorf("事件%s(%s)不应改变部件数量", ev.ID, ev.Kind)
            }
        default:
            return fmt.Errorf("事件%s类型未知: %s", ev.ID, ev.Kind)
        }
        kindsSeen[ev.Kind] = true
        for _, mv := range ev.Moves {
            if !componentOf[mv.Component] {
                return fmt.Errorf("事件%s引用了未登记部件: %s", ev.ID, mv.Component)
            }
            if mv.Quantity < 1 || mv.From == "" || mv.To == "" || mv.From == mv.To {
                return fmt.Errorf("事件%s流转记录不完整: %+v", ev.ID, mv)
            }
            if balance[mv.Component] == nil {
                balance[mv.Component] = map[string]int{}
            }
            stock := balance[mv.Component]
            if additiveKinds[ev.Kind] {
                stock[mv.To] += mv.Quantity
            } else {
                stock[mv.From] -= mv.Quantity
                if stock[mv.From] < 0 {
                    return fmt.Errorf("事件%s使%s在%s的数量为负，部件来源与数量不平衡", ev.ID, mv.Component, mv.From)
                }
                stock[mv.To] += mv.Quantity
            }
        }
        if err := checkRefs(ev, batches, reports, standardRefs); err != nil {
            return err
        }
        switch ev.Kind {
        case "批次登记", "补货":
            batches[ev.Refs.Batch] = true
        case "检验", "复检", "实验室更正":
            reports[ev.Refs.Report] = true
        }
    }
    for kind := range additiveKinds {
        if !kindsSeen[kind] {
            return fmt.Errorf("证据链缺少%s事件", kind)
        }
    }
    for kind := range conservingKinds {
        if !kindsSeen[kind] {
            return fmt.Errorf("证据链缺少%s事件", kind)
        }
    }
    for kind := range evidenceKinds {
        if !kindsSeen[kind] {
            return fmt.Errorf("证据链缺少%s事件", kind)
        }
    }
    return nil
}

// checkRefs 按事件类型检查引用，保证批次、报告、标准与更正关系可追溯。
func checkRefs(ev ChainEvent, batches, reports, standardRefs map[string]bool) error {
    refs := ev.Refs
    needStandard := func() error {
        if !standardRefs[refs.Standard] {
            return fmt.Errorf("事件%s引用了未登记的标准版本: %s", ev.ID, refs.Standard)
        }
        return nil
    }
    switch ev.Kind {
    case "合同登记":
        if refs.Contract == "" || refs.StyleNo == "" {
            return fmt.Errorf("事件%s缺少合同编号或款号", ev.ID)
        }
    case "批次登记", "补货":
        if refs.Batch == "" {
            return fmt.Errorf("事件%s缺少生产批次号", ev.ID)
        }
        if ev.Kind == "批次登记" && len(refs.Materials) == 0 {
            return fmt.Errorf("事件%s缺少面辅料登记", ev.ID)
        }
    case "抽样封签":
        if len(refs.Seals) == 0 {
            return fmt.Errorf("事件%s缺少封签号", ev.ID)
        }
    case "检验", "复检":
        if refs.Report == "" {
            return fmt.Errorf("事件%s缺少检验报告编号", ev.ID)
        }
        if err := needStandard(); err != nil {
            return err
        }
    case "实验室更正":
        if refs.Report == "" || refs.Corrects == "" {
            return fmt.Errorf("事件%s缺少更正报告编号或被更正报告", ev.ID)
        }
        if !reports[refs.Corrects] {
            return fmt.Errorf("事件%s更正的报告不存在: %s", ev.ID, refs.Corrects)
        }
    case "冻结":
        if !batches[refs.Batch] {
            return fmt.Errorf("事件%s冻结的批次未登记: %s", ev.ID, refs.Batch)
        }
    case "通知":
        if len(refs.NotifyTargets) == 0 {
            return fmt.Errorf("事件%s缺少通知对象", ev.ID)
        }
    case "结算索赔":
        hasStandard, hasReport := false, false
        for _, item := range ev.Evidence {
            if standardRefs[item] {
                hasStandard = true
            }
            if reports[item] {
                hasReport = true
            }
        }
        if !hasStandard || !hasReport {
            return fmt.Errorf("事件%s的结算索赔必须同时引用标准版本与检验原件", ev.ID)
        }
    }
    return nil
}
