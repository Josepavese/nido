# Nido Configuration

Copy the example and edit:

```bash
cp config.example.env ~/.nido/config.env
```

Nido loads `~/.nido/config.env` by default.

### Modern Keys

- `BACKUP_DIR`: Directory for VM backups
- `TEMPLATE_DEFAULT`: Default template for `spawn`
- `SSH_USER`: Default SSH user (defaults to `vmuser`)
- `IMAGE_DIR`: Directory for downloaded images
- `LINKED_CLONES`: Enabled by default (space saving)

Override config location with:

```bash
NIDO_CONFIG=/path/to/config.env nido <command>
```
