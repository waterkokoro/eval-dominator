export const datasetTypeOptions = [
  { value: "opencompass_demo", label: "OpenCompass Demo" },
  { value: "opencompass_standard", label: "OpenCompass 标准数据集" },
  { value: "custom", label: "自定义数据集" }
];

export const datasetTypeText = datasetTypeOptions.reduce((acc, item) => {
  acc[item.value] = item.label;
  return acc;
}, {});

export function getDatasetTypeText(type) {
  return datasetTypeText[type] || type || "未知数据集";
}
