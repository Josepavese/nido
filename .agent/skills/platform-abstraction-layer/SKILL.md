---
name: platform-abstraction-layer
description: Enforces the mandatory Platform Abstraction Layer (Layer/Middleware/Stack) architecture. Use when implementing or modifying any logic that interacts with the Operating System, requires cross-platform support (Windows, Linux, macOS), or when designing core system components.
---

# Platform Abstraction Layer

This skill enforces a mandatory architectural pattern to ensure Nido remains **Agnostic**, **Multiplatform**, and **Modular**.

## The Agnostic Principle

**The Logic Layer must be Agnostic.**
Core business logic (the "Upper Layers") should never know or care which Operating System it is running on. It should never import `syscall` or check `runtime.GOOS` directly. Instead, it must ask a High-Level Interface to perform a semantic action (e.g., "Get Home Directory"), trusting the underlying layers to handle the implementation.

## The Middleware Pattern (Layer/Middleware/Stack)

All system interactions must traverse three distinct layers. This separation is **MANDATORY**.

1.  **Logic Layer (The "What")**: The pure business logic. It asks for high-level operations.
    *   *Constraint*: Zero OS-specific imports. Zero `runtime.GOOS` checks.
2.  **Middleware/Abstraction Layer (The "How it adapts")**: The translation layer. It receives the agnostic request and decides *which* provider or strategy to use based on the current environment.
    *   *Function*: Intermediary routing, standardization, and fallback logic.
3.  **Stack/Provider Layer (The "How it works")**: The OS-specific implementation.
    *   *Constraint*: strictly scoped to a single platform or technology (e.g., `linux_fs`, `windows_registry`).

### Implementation Strategy

When you need to access OS functionality (Filesystem, Network, Process Execution, hardware):

1.  **Define the Interface**: Create an agnostic interface in the Middleware layer (e.g., `FileSystem`).
2.  **Implement Providers**: Create implementations in the Stack layer (e.g., `LinuxFileSystem`, `WindowsFileSystem`).
3.  **Route in Middleware**: The Middleware initializes the correct provider on startup.

### Single Source of Truth

This architecture relies heavily on centralized configuration and constants.
**ALWAYS** adhere to the **Single Source of Truth** skill when defining paths, defaults, and configuration keys.
Refer to: `@[.agent/skills/single-source-of-truth/SKILL.md]`

## Examples

### Correct Agnostic delegation

```go
// LOGIC LAYER (Agnostic)
// User wants to save config. We don't ask "which OS?". We just ask "Where is the config home?"
path, err := system.GetConfigHome() 

// MIDDLEWARE LAYER (Router)
func GetConfigHome() (string, error) {
    // Delegates to the loaded provider for the current runtime
    return currentProvider.ConfigHome(), nil
}

// STACK LAYER (Linux Provider)
func (l *LinuxProvider) ConfigHome() string {
    return os.Getenv("XDG_CONFIG_HOME") // standard linux
}

// STACK LAYER (Windows Provider)
func (w *WindowsProvider) ConfigHome() string {
    return os.Getenv("APPDATA") // standard windows
}
```

### Incorrect (Violates Agnostic Principle)

```go
// BAD: Logic layer knows too much!
func SaveConfig() {
    if runtime.GOOS == "windows" {
       // ... windows logic
    } else {
       // ... linux logic
    }
}
```

## References

*   **Architecture & Diagrams**: See `references/architecture.md` for visual flows and strict layering rules.
