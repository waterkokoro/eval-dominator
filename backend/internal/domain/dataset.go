package domain

import "time"

type DatasetSource string

const (
	DatasetSourceBuiltin     DatasetSource = "builtin"
	DatasetSourceCustom      DatasetSource = "custom"
	DatasetSourceHuggingFace DatasetSource = "huggingface"
)

type Dataset struct {
	ID                int64
	Code              string
	DisplayName       string
	Description       string
	Type              string
	Source            DatasetSource
	SampleCount       int
	Enabled           bool
	InferenceMode     string // 推理方式：gen / ppl / "" (未知)
	ConfigPath        string
	ExtraJSON         string
	HuggingFaceRepo   string // HF 仓库地址 e.g. "openai/gsm8k"
	HuggingFaceSubset string // HF 子集
	LocalPath         string // 本地存储路径（HF下载后或自定义文件路径）
	FileFormat        string // csv / jsonl / json
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// IsRemoteCompatible 判断该数据集能否被远程 API 模型（OpenAISDK 等）评测。
// PPL 模式需要直接读 logits，仅本地 HuggingFace 模型支持。
func (d Dataset) IsRemoteCompatible() bool {
	return d.InferenceMode != "ppl"
}
