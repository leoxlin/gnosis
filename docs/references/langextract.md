---
type: Reference
title: LangExtract
description: A Python library from Google that uses LLMs to extract structured information from unstructured text with source grounding, schema constraints, and interactive visualization.
resource: https://github.com/google/langextract
tags: [python, llm, extraction, structured-output, google, nlp]
timestamp: 2026-07-09T03:31:48Z
---

# LangExtract

LangExtract is a Python library published by Google that uses large language models to extract structured entities from unstructured text. It is designed for long documents (clinical notes, reports, novels) and emphasizes source-grounded output, controlled generation, and reviewability.

## Key capabilities

| Capability | What it does |
|---|---|
| **Source grounding** | Every extraction is mapped back to its exact character interval in the input text. Unlocatable extractions are flagged with `char_interval = None`. |
| **Schema-constrained output** | Few-shot examples define extraction classes, attributes, and structure; supported models use controlled generation to enforce the schema. |
| **Long-document processing** | Text is chunked and processed in parallel across multiple extraction passes to improve recall on large inputs. |
| **Interactive visualization** | Results can be rendered as a self-contained HTML file for reviewing extractions in their original context. |
| **Flexible model support** | Works with Google Gemini, OpenAI models, and local models via Ollama; custom providers can be registered via a plugin system. |

## Core workflow

1. Define a prompt describing what to extract.
2. Provide one or more `ExampleData` objects with verbatim extractions from example text.
3. Call `lx.extract(...)` with the input text or document URL.
4. Save results as JSONL and optionally generate an HTML visualization.

## Relevant design ideas

- **Grounding by default**: The library detects when the model extracts content from few-shot examples rather than the input text, making it easier to filter hallucinated or ungrounded results.
- **Example-driven schemas**: Rather than a formal schema language, the output structure is implied by the shape of the provided examples.
- **Parallel, multi-pass extraction**: For long documents, increasing `extraction_passes` and `max_workers` trades compute for recall.

## Related references

- [Obsidian Wiki (Ar9av)](./obsidian-wiki.md) — another agent-driven knowledge-extraction framework.
- [Karpathy LLM Wiki pattern](./karpathy-llm-wiki.md) — the broader pattern of maintaining a structured wiki from raw sources.

## Citations

[1] [LangExtract repository](https://github.com/google/langextract)
