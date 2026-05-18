---
name: "image-chroma-remove"
description: "Use when an image has a flat chroma-key background that should become real transparency, especially for pixel-art sprites generated on green, magenta, cyan, or another solid removable background. Supports explicit key colors, sampled border/corner keys, edge cleanup, alpha validation, and optional nearest-neighbor resizing for exact sprite dimensions."
---

# Image Chroma Remove

Convert a flat chroma-key background into real alpha transparency. This is useful after generating pixel-art icons or sprites on a solid key color because the image model cannot always produce true transparent PNGs directly.

## Quick Start

Use the bundled script:

```bash
export CODEX_HOME="${CODEX_HOME:-$HOME/.codex}"
export CHROMA_REMOVE="$CODEX_HOME/skills/image-chroma-remove/scripts/remove_chroma.py"
python3 "$CHROMA_REMOVE" --input source.png --out sprite.png --auto-key border --soft-matte --despill
```

For exact pixel-art output size:

```bash
python3 "$CHROMA_REMOVE" \
  --input source.png \
  --out r_sticks.png \
  --auto-key border \
  --soft-matte \
  --despill \
  --resize 64x64 \
  --resample nearest
```

For a non-green key:

```bash
python3 "$CHROMA_REMOVE" --input source.png --out sprite.png --key-color "#ff00ff" --soft-matte --despill
```

## Workflow

1. Choose a key color that does not appear in the subject. Common choices: `#00ff00`, `#ff00ff`, `#00ffff`.
2. Generate or provide an image with a perfectly flat key background: no shadows, gradients, checkerboards, floor plane, or texture.
3. Prefer `--auto-key border` when the background reaches the image edges; use `--auto-key corners` when the subject may touch borders but corners are clear.
4. Use `--key-color "#rrggbb"` when the intended key color is known and generation preserved it closely.
5. For pixel art, include `--resize WIDTHxHEIGHT --resample nearest` only after alpha removal.
6. Validate output with `--validate` or inspect transparent corners before using the sprite in a project.

## Script Options

Core options:

- `--input`: source image with chroma-key background.
- `--out`: output `.png` or `.webp` with alpha.
- `--key-color "#rrggbb"`: explicit key color.
- `--auto-key none|corners|border`: sample the key color from the image.
- `--tolerance`: hard key threshold for exact removal.
- `--soft-matte`: create partially transparent edge pixels between thresholds.
- `--transparent-threshold` and `--opaque-threshold`: tune soft matte.
- `--edge-contract`: remove a thin fringe by shrinking alpha.
- `--edge-feather`: soften stair-stepped edges only when appropriate.
- `--despill`: reduce key-color tint on edge pixels.
- `--connected-background`: remove only key-like pixels connected to the image border. Use this for noisy generated chroma backgrounds so isolated purple/green/blue details inside the subject are preserved.
- `--connected-tolerance`: tolerance for `--connected-background`; defaults to `--tolerance`.
- `--flatten-background-out PATH`: also write a copy where connected background pixels are replaced with one exact key color. Use this when you want an auditable flattened raw source before exact removal.
- `--flatten-disconnected-threshold N`: while writing the flattened source, also flatten disconnected pixels within `N` color distance of the key. Use this for small isolated background remnants after border-connected flattening.
- `--resize WIDTHxHEIGHT`: write an exact output size.
- `--resample nearest|lanczos`: use `nearest` for pixel art.
- `--trim`: crop to the alpha bounding box before optional resize.
- `--pad N`: add transparent padding after trimming.
- `--validate`: fail if the output has no alpha, opaque corners, or too little visible subject.
- `--force`: overwrite an existing output.

## Pixel-Art Defaults

For generated sprite icons, use:

```bash
python3 "$CHROMA_REMOVE" \
  --input chroma_source.png \
  --out sprite.png \
  --auto-key border \
  --soft-matte \
  --transparent-threshold 12 \
  --opaque-threshold 220 \
  --despill \
  --resize 64x64 \
  --resample nearest \
  --validate \
  --force
```

If the edge still has a visible color fringe, retry once with `--edge-contract 1`.

For noisy generated backgrounds, prefer a border-connected hard mask before trying soft matte:

```bash
python3 "$CHROMA_REMOVE" \
  --input chroma_source.png \
  --out sprite.png \
  --key-color "#ff00ff" \
  --connected-background \
  --connected-tolerance 32 \
  --resize 64x64 \
  --resample nearest \
  --validate \
  --force
```

This removes only key-like pixels reachable from the image border. It is safer than broad global tolerance when the subject contains similar colors.

To create a flattened chroma source for audit or a second exact-removal pass:

```bash
python3 "$CHROMA_REMOVE" \
  --input chroma_source.png \
  --out preview_alpha.png \
  --auto-key border \
  --connected-background \
  --connected-tolerance 50 \
  --flatten-background-out chroma_flattened.png \
  --force
```

For generated atlases with small disconnected background flecks, add:

```bash
  --flatten-disconnected-threshold 20
```

Then run a second pass on `chroma_flattened.png` with exact or low-tolerance removal.

## Dependencies

Requires Pillow:

```bash
python3 -m pip install pillow
```

When available, prefer the bundled workspace Python because it usually already has Pillow installed.
