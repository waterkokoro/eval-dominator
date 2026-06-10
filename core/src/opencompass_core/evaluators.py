"""自定义 evaluator，注册到 OpenCompass ICL_EVALUATORS registry。"""

import json as _json

from opencompass.openicl.icl_evaluator import BaseEvaluator
from opencompass.registry import ICL_EVALUATORS


@ICL_EVALUATORS.register_module()
class KeywordMatchEvaluator(BaseEvaluator):
    """关键词命中率评分：检测模型输出是否包含期望的关键词。

    继承 BaseEvaluator 由父类提供 evaluate(k, n, dataset, **score_kwargs) 入口，
    子类只需实现 score(predictions, references) 即可。
    """

    def __init__(self, keyword_column: str = "expected_keywords", **kwargs):
        super().__init__(**kwargs)
        self.keyword_column = keyword_column

    def score(self, predictions, references):
        """OpenCompass BaseEvaluator.evaluate 会按 self.score 的签名挑参数传入，
        因此这里只声明 predictions / references 两个核心参数即可。

        注意：这里不返回 details——一旦 score 自带 details，BaseEvaluator.evaluate
        会走 group() 路径并要求 dataset 中存在 subdivision/idx 字段（仅 LiveCodeBench
        等特殊数据集才有）。返回纯 metrics dict 后，openicl_eval.py 会自行 format。
        """
        hits = 0
        total = len(predictions)
        for pred, ref in zip(predictions, references):
            # 解析 references：可能是 JSON 字符串、列表，或单字符串
            keywords: list = []
            if isinstance(ref, str):
                try:
                    parsed = _json.loads(ref)
                    if isinstance(parsed, list):
                        keywords = parsed
                    else:
                        keywords = [str(parsed)]
                except Exception:
                    keywords = [ref]
            elif isinstance(ref, list):
                keywords = ref
            elif ref is not None:
                keywords = [str(ref)]

            pred_str = pred or ""
            if not keywords or not pred_str:
                continue
            pred_lower = pred_str.lower()
            if any(str(kw).lower() in pred_lower for kw in keywords):
                hits += 1

        keyword_hit_rate = round(hits / max(total, 1) * 100, 2)
        return {
            "keyword_hit_rate": keyword_hit_rate,
            "accuracy": keyword_hit_rate,  # 兼容上层按 accuracy 取分的展示
        }
