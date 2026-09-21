package schooluniformquality

import (
    "bytes"
    "encoding/json"
    "errors"
)

// Record 表示项目共享的领域资料。
type Record struct {
    Domain string `json:"domain"`
    Version int `json:"version"`
    SampleID string `json:"sample_id"`
    Actors []string `json:"actors"`
    Facts []string `json:"facts"`
    EvidenceChain []string `json:"evidence_chain"`
    Operations []string `json:"operations"`
    Constraints []string `json:"constraints"`
    AccessRules []string `json:"access_rules"`
}

// Parse 读取并检查带版本的业务资料。
func Parse(raw []byte) (Record, error) {
    var value Record
    decoder := json.NewDecoder(bytes.NewReader(raw))
    decoder.DisallowUnknownFields()
    if err := decoder.Decode(&value); err != nil { return Record{}, err }
    if value.Domain == "" || value.Version < 1 || value.SampleID == "" ||
        len(value.Actors) < 2 || len(value.Facts) < 2 ||
        len(value.EvidenceChain) < 2 || len(value.Operations) < 1 ||
        len(value.Constraints) < 2 || len(value.AccessRules) < 1 {
        return Record{}, errors.New("共享资料缺少必要字段")
    }
    return value, nil
}
