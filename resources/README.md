# Nido visual assets

Keep the retro workstation / hatching egg identity, readable at small sizes.
The README introduces the product; diagrams explain behavior rather than making
performance or isolation promises.

| Asset | Consumer / purpose |
| --- | --- |
| `nido-hero.webp` | README cover, 1440 × 720; compressed WebP |
| `nido-workflow.svg` | Editable workflow infographic with title/description and matching README alt text |
| `nido_social_preview.png` | 1280 × 640 social preview, indexed PNG; upload through repository settings when needed |
| `nido.png` | 512 × 512 installer / desktop icon; preserve this PNG path contract |
| `nido-icon.png` | Same icon bytes at the MCP Registry's existing PNG URL |

The hero was generated with the built-in Imagegen tool, then resized and encoded
for web delivery. Prompt direction: **NIDO, Real VMs. Agent-ready.; a luminous
circuit egg in a wire nest inside a retro workstation, with three separate VM
modules; dark navy, cyan and coral, restrained violet, generous margins, no
performance claims or simulated UI.** The SVG is hand-authored source.

Superseded square marketing panels and orphan screenshots were retired from the
active tree. Their originals remain in Git history. Do not present an invented
dashboard as a product screenshot. For a future UI screenshot, capture a current
disposable environment without private VM names or customer data.

Optimize delivered dimensions and encoding before adding raster files. PNG icon
paths are consumed by installers and registry metadata; WebP is appropriate for
editorial illustrations. Keep text alternatives in the consuming Markdown.
