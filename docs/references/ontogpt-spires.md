---
type: Reference
title: OntoGPT (SPIRES)
subtitle: Structured Prompt Interrogation and Recursive Extraction of Semantics
description: A Python package and zero-shot knowledge extraction method that uses large language models, instruction prompts, and ontology-based grounding to populate structured schemas from unstructured text.
resource: https://github.com/monarch-initiative/ontogpt/
tags: [python, llm, extraction, ontology, grounding, nlp, biomedical, zero-shot, knowledge-base]
timestamp: 2026-07-09T03:33:17Z
---

# OntoGPT (SPIRES)

**OntoGPT** is a Python package from the Monarch Initiative for extracting structured information from text with large language models (LLMs), instruction prompts, and ontology-based grounding. It implements **SPIRES** (Structured Prompt Interrogation and Recursive Extraction of Semantics), a zero-shot method for populating complex, nested knowledge schemas from unstructured text without task-specific training data.

## Core idea

Instead of training a dedicated relation-extraction model, SPIRES relies on the zero-shot query-answering and structured-output capabilities of GPT-3+ class models. Given:

1. a detailed, user-defined schema (classes, slots, and expected value types), and
2. an input text,

SPIRES recursively interrogates the LLM with targeted prompts until the returned data conform to the schema. Extracted entities are then mapped to identifiers from existing ontologies and vocabularies via the Ontology Access Kit (OAK).

## Key capabilities

| Capability | What it does |
|---|---|
| **Zero-shot schema extraction** | No task-specific training examples are required; the model is prompted with the schema definition itself. |
| **Recursive prompting** | Complex or nested slots are filled by issuing follow-up prompts until the full structure is populated. |
| **Ontology grounding** | Extracted terms are linked to identifiers from public ontologies and controlled vocabularies. |
| **Structured extraction** | Extracts typed entities and relations from free text using LLMs guided by instruction prompts. |
| **Multi-provider LLM support** | Uses LiteLLM to interface with OpenAI, Anthropic, Mistral, Groq, Azure OpenAI, Ollama, and other providers. |
| **Web interface** | Includes an optional minimal web app (`web-ontogpt`) for running extractions in a browser. |
| **Flexible domains** | Demonstrated on food recipes, cellular signaling pathways, disease treatments, multi-step drug mechanisms, and chemical-disease causation graphs. |

## Core workflow

1. Install `ontogpt` with `pip install ontogpt`.
2. Set the API key for the desired provider (e.g., `OPENAI_API_KEY`).
3. Run `ontogpt extract -i <input> -t <template>` to extract structured objects.
4. Review the `extracted_object` output, with terms grounded to ontologies where possible.

## Trade-offs

* **Accuracy** is comparable to mid-range supervised relation-extraction methods — better than fully naive prompting, but below top-performing task-specific models.
* **Customizability** is high because the extraction target is just a schema, not a trained model.
* **Cost and latency** scale with the number of recursive LLM calls needed to fill complex schemas.

## Relevant design ideas

- **Schema-driven extraction**: The output contract is a structured schema rather than a fixed ontology or flat entity list.
- **Grounding in public vocabularies**: Extracted concepts carry stable identifiers, making them usable outside the LLM context.
- **Prompting as an extraction runtime**: For new tasks, the "model" is replaced by a prompt program, reducing the barrier to new domains.
- **Provider abstraction via LiteLLM**: The same extraction pipeline works across many model backends by relying on LiteLLM for model routing and credential handling.

## Related references

- [LangExtract](./langextract.md) — another LLM-based structured-extraction library with source grounding.
- [LLM Wiki (Karpathy)](./karpathy-llm-wiki.md) — the broader pattern of maintaining a structured knowledge base from raw sources.
- [Open Knowledge Format (OKF)](./okf.md) — a format for representing structured knowledge extracted from text.

## Citations

[1] [OntoGPT repository](https://github.com/monarch-initiative/ontogpt/)

[2] Caufield JH, Hegde H, Emonet V, Harris NL, Joachimiak MP, Matentzoglu N, et al. Structured prompt interrogation and recursive extraction of semantics (SPIRES): A method for populating knowledge bases using zero-shot learning. *Bioinformatics*, Volume 40, Issue 3, March 2024, btae104. https://doi.org/10.1093/bioinformatics/btae104
