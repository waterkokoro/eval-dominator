import i18n from "@/locales";

export const DATASET_TYPE_KEYS = [
  "opencompass_demo",
  "opencompass_standard",
  "custom",
  "huggingface"
];

export const DATASET_SOURCE_KEYS = [
  "builtin",
  "custom",
  "huggingface"
];

export const TASK_TYPE_OPTIONS = [
  { value: "choice", labelKey: "dataset.dialog.fields.taskTypeChoice" },
  { value: "qa", labelKey: "dataset.dialog.fields.taskTypeQA" },
  { value: "classification", labelKey: "dataset.dialog.fields.taskTypeClassification" }
];

export function getDatasetTypeText(type) {
  if (!type) return i18n.t("dataset.type.unknown");
  if (DATASET_TYPE_KEYS.includes(type)) {
    return i18n.t(`dataset.type.${type}`);
  }
  return type;
}

export function getDatasetTypeOptions() {
  return DATASET_TYPE_KEYS.map((value) => ({
    value,
    label: i18n.t(`dataset.type.${value}`)
  }));
}

export function getDatasetSourceOptions() {
  return DATASET_SOURCE_KEYS.map((value) => ({
    value,
    label: i18n.t(`dataset.source.${value}`)
  }));
}
