import i18n from "@/locales";

export const DATASET_TYPE_KEYS = [
  "opencompass_demo",
  "opencompass_standard",
  "custom"
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
