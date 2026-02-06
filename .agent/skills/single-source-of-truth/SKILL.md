---
name: single-source-of-truth
description: Maniacally consolidate parameters, configurations, and data into a Single Source of Truth. Favor YAML/JSON files over hardcoded constants. Use when asked to 'centralize' data, constants, or establish a 'Single Source of Truth'.
---

# Single Source of Truth

This skill enforces a "maniacal" approach to data and configuration centralization. The goal is to eliminate hardcoded values and redundant definitions by establishing or using existing Single Sources of Truth.

## Core Principles

1. **Maniacal Research First**: Never create a new configuration location without first exhaustively checking if one already exists that could host the data.
2. **Single Source of Truth Supremacy**: There must be exactly one place where a piece of information originates.
3. **Externalize Configuration**: Prefer YAML or JSON files for settings. Avoid hardcoding values in the code that might change between environments or versions.
4. **Information Flow**: Ensure that the centralized data is correctly propagated to all consumers (injected, loaded via config, etc.).

## Workflow

When asked to centralize something, follow these steps:

1. **Exhaustive Search**: Scan the entire codebase for:
    - Files named `*config*`, `*settings*`, `*params*`, `*version*`.
    - Directories like `config/`, `settings/`, `assets/config/`.
    - Existing patterns (e.g., a `domain/version.go` file used to sync other files).
2. **Analyze & Reuse**: If a suitable central location exists, evaluate if it makes sense to extend it. Reusing a well-established pattern is better than creating a fragment.
3. **Create (If Necessary)**: If no suitable location exists, create one.
    - **Format Choice**: YAML (preferred for readability) > JSON > Go/Python Constants (last resort, only if needed for build-time).
4. **Migration**: Replace all hardcoded occurrences with references to the new Single Source of Truth.
5. **Synchronization**: If the Single Source of Truth needs to be reflected in multiple files (e.g., a version number in `codetree.md` and `README.md`), ensure a synchronization mechanism is in place.

For a detailed walkthrough of the "maniacal" process, see [maniacal-workflows.md](references/maniacal-workflows.md).
