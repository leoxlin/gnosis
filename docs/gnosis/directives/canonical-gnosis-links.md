---
type: GnosisDirective
title: Canonical `gnosis` links
description: Read and render canonical vault-qualified gnosis links, with `core` as the bundled vault.
status: done
---

# Goal

Make canonical `gnosis://<vault-name>/<path>` links usable throughout the CLI and render resolved internal document links in that form.

# Scope

Update page identity generation, URI selection, `read` argument handling, Markdown rendering, and text concepts output. Do not convert external URLs, unresolved links, or non-document assets.

# Dependencies

- [Define `gnosis` URI format](../decisions/define-gnosis-uri-format.md)

# Implementation plan

1. Add failing vault and CLI tests for canonical URI generation, positional `read` URIs, and rendered internal links in read and concepts output.
2. Change URI construction and selection to emit and resolve only the canonical authority-based form.
3. Add Markdown link rendering that resolves effective internal document destinations and substitutes their canonical URIs without changing external, unresolved, or asset links.
4. Make every `read` mode return rendered Markdown and include canonical links in text concepts output.
5. Format the changed Go files, run focused and full test suites, inspect the diff, and set this directive to `done` only after each acceptance criterion has fresh evidence.

# Acceptance criteria

- `gnosis read gnosis://<vault-name>/<path>` reads an exact effective page.
- The bundled documentation and concepts identify their source as `gnosis://core/<path>`.
- Every resolved internal Markdown document link emitted by `gnosis read` uses its canonical `gnosis://` URI; external, unresolved, and asset links remain unchanged.
- Both `gnosis concepts` and `gnosis concepts -type <type>` text output render links to their records as canonical `gnosis://` URIs.
- Document selection uses only canonical `gnosis://` URIs.
- Focused and full Go tests pass.
