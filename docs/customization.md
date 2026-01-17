# Customizing Nido

Nido supports custom themes via a JSON configuration file. You can define your own color palettes to match your terminal rice or personal preference.

## Theme Location

Nido looks for a `themes.json` file in:

- `~/.nido/themes.json` (Recommended)

## Schema

The `themes.json` file should contain an object with a `themes` array. Each theme has a `name` and a `palette` object.

Colors are defined using `adaptive colors`, meaning you provide a `light` value (for light terminal backgrounds) and a `dark` value (for dark terminal backgrounds). The TUI automatically selects the correct one.

### Example `themes.json`

```json
{
  "themes": [
    {
      "name": "Dracula",
      "mode": "dark",
      "palette": {
        "background":       { "light": "#282A36", "dark": "#282A36" },
        "surface":          { "light": "#44475A", "dark": "#44475A" },
        "surface_subtle":   { "light": "#6272A4", "dark": "#6272A4" },
        "surface_highlight":{ "light": "#BD93F9", "dark": "#BD93F9" },
        
        "text":             { "light": "#F8F8F2", "dark": "#F8F8F2" },
        "text_dim":         { "light": "#6272A4", "dark": "#6272A4" },
        "text_muted":       { "light": "#6272A4", "dark": "#6272A4" },
        
        "accent":           { "light": "#BD93F9", "dark": "#BD93F9" },
        "accent_strong":    { "light": "#FF79C6", "dark": "#FF79C6" },
        
        "success":          { "light": "#50FA7B", "dark": "#50FA7B" },
        "warning":          { "light": "#F1FA8C", "dark": "#F1FA8C" },
        "error":            { "light": "#FF5555", "dark": "#FF5555" },
        
        "focus":            { "light": "#BD93F9", "dark": "#BD93F9" },
        "hover":            { "light": "#44475A", "dark": "#44475A" },
        "disabled":         { "light": "#6272A4", "dark": "#6272A4" }
      }
    }
  ]
}
```

## Applying a Theme

1. Create the file at `~/.nido/themes.json`.
2. Restart Nido.
3. Navigate to **SYSTEM** > **Rice**.
4. Select your theme from the dropdown list.
5. Press **SAVE**.
