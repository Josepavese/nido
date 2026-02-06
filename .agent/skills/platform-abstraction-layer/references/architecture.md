# Layer / Middleware / Stack Architecture

This document details the mandatory architectural flow for Cross-Platform Nido components.

## Conceptual Flow

```mermaid
graph TD
    User[User / CLI Command] --> Logic[Logic Layer<br/>(Agnostic Business Logic)]
    Logic -- "Semantic Call\n(e.g. OpenUrl)" --> Middleware[Middleware/Abstraction<br/>(Interface Router)]
    Middleware -- "Selects Implementation" --> Stack{Stack Provider}
    Stack -->|Linux| LinuxProvider[Linux Implementation]
    Stack -->|Windows| WinProvider[Windows Implementation]
    Stack -->|macOS| MacProvider[Darwin Implementation]
    
    LinuxProvider --> OS[Operating System]
    WinProvider --> OS
    MacProvider --> OS
```

## Layers Defined

### 1. Logic Layer
*   **Responsibility**: Execute user intent. Manage state. Orchestrate high-level workflows.
*   **Knowledge**: Knows **WHAT** needs to be done.
*   **Ignorance**: Knows **NOTHING** about the OS or specific driver implementation.
*   **Example**: `vm.Start()`

### 2. Middleware (Abstraction) Layer
*   **Responsibility**: Interface definition. Runtime detection. Dispatching.
*   **Knowledge**: Knows available providers and the running environment.
*   **Action**: `GetVMProvider() -> returns *QEMUProvider` or `*HyperKitProvider`.
*   **Example**: `sysutil.Open(url)` (checks OS, calls `exec("xdg-open")` or `exec("cmd /c start")`)

### 3. Stack (Provider) Layer
*   **Responsibility**: The dirty work. Syscalls. OS-specific API calls. Flag formatting.
*   **Knowledge**: Knows deeply about ONE platform.
*   **Isolation**: Should be in files with build tags (e.g., `_linux.go`, `_windows.go`).
*   **Example**: `func (p *LinuxProvider) BindVFIO(...)`

## Guidelines

1.  **Isolation**: If you change the Windows implementation, the Logic layer code should not even need recompilation (conceptually).
2.  **Interfaces**: Middleware defines the Interface. Stack implements it. Logic consumes it.
3.  **SSOT**: Configuration for "Which provider to use" comes from the Single Source of Truth configuration system.
