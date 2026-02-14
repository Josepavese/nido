---
description: "how to install Nido locally and setup Bash completion"
---
// turbo-all

1. Navigate to the project root and build the binaries:

```bash
go build -o nido ./cmd/nido
go build -o nido-validator ./cmd/nido-validator
```

2. Ensure the local installation directory exists:

```bash
mkdir -p ~/.nido/bin
mkdir -p ~/.nido/registry
```

3. Copy the fresh binaries and registry to the installation directory:

```bash
cp nido nido-validator ~/.nido/bin/
cp -r registry/* ~/.nido/registry/
```

4. Generate and install the Bash completion script:

```bash
~/.nido/bin/nido completion bash > ~/.nido/bin/nido.bash
```

5. Ensure your `~/.bashrc` sources the completion script:

```bash
grep -q "source ~/.nido/bin/nido.bash" ~/.bashrc || echo 'source ~/.nido/bin/nido.bash' >> ~/.bashrc
```

6. Refresh your current shell session:

```bash
source ~/.bashrc
```

7. Verify the installation:

```bash
nido version
```
