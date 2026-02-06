# Maniacal Centralization Long-form Workflow

This reference guide details the step-by-step process for achieving perfect centralization.

## 1. The Research Phase (Maniacal Scanning)

Before touching any code, you must know the landscape.

- **Grep for Hardcoded Values**: If you are centralizing a specific value (e.g., an API endpoint), grep for it everywhere to identify all consumers.
- **Identify Config Patterns**: Look for how other things are configured.
  - `find . -maxdepth 3 -name "*config*"`
  - `grep -r "json.Unmarshal" .`
  - `grep -r "yaml.Unmarshal" .`
- **Dependency Mapping**: Visualize how the data flows. Does the `main.go` load it and pass it down? Or do modules read it themselves?

## 2. Choosing the Format

Follow this hierarchy:

1. **YAML**: Best for human-readable configurations, comments, and structure.
2. **JSON**: Best for machine-interop or if the project already heavily uses JSON.
3. **Dedicated Configuration Module**: If the language requires it (like Go `internal/config`), create a struct-based loader that reads from YAML/JSON.

## 3. The Implementation Flow

### Step A: Define the Schema

Declare the structure clearly. If using YAML/JSON, create a sample or a schema.

### Step B: The Loader

Implement a robust loader that handles:

- Default values.
- Environment variable overrides.
- Missing file errors (with descriptive messages).

### Step C: Injection

Inject the configuration into the components that need it. Avoid global variables where possible; favor dependency injection.

## 4. Maniacal Verification

1. **Dry Run**: Print the loaded config to verify correctness.
2. **Consistency Check**: Ensure no hardcoded values remain.
3. **Environment Parity**: Is this config flexible enough for dev/prod?
