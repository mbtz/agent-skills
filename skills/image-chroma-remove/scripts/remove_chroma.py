#!/usr/bin/env python3
"""Remove a flat chroma-key background and write an alpha image.

Designed for generated pixel-art sprites: remove green/magenta/cyan/etc.
backgrounds, optionally trim, pad, and resize with nearest-neighbor sampling.
"""

from __future__ import annotations

import argparse
from collections import deque
from pathlib import Path
import re
from statistics import median
import sys
from typing import Tuple


Color = Tuple[int, int, int]
KEY_DOMINANCE_THRESHOLD = 16.0
ALPHA_NOISE_FLOOR = 8


def die(message: str, code: int = 1) -> None:
    print(f"Error: {message}", file=sys.stderr)
    raise SystemExit(code)


def load_pillow():
    try:
        from PIL import Image, ImageFilter, ImageOps
    except ImportError:
        die("Pillow is required. Install it with `python3 -m pip install pillow`.")
    return Image, ImageFilter, ImageOps


def parse_key_color(raw: str) -> Color:
    match = re.fullmatch(r"#?([0-9a-fA-F]{6})", raw.strip())
    if not match:
        die("key color must be a hex RGB value like #00ff00.")
    value = match.group(1)
    return (int(value[0:2], 16), int(value[2:4], 16), int(value[4:6], 16))


def parse_size(raw: str | None) -> tuple[int, int] | None:
    if raw is None:
        return None
    match = re.fullmatch(r"([1-9][0-9]*)x([1-9][0-9]*)", raw.strip().lower())
    if not match:
        die("--resize must use WIDTHxHEIGHT, for example 64x64.")
    return (int(match.group(1)), int(match.group(2)))


def channel_distance(a: Color, b: Color) -> int:
    return max(abs(a[0] - b[0]), abs(a[1] - b[1]), abs(a[2] - b[2]))


def clamp_channel(value: float) -> int:
    return max(0, min(255, int(round(value))))


def smoothstep(value: float) -> float:
    value = max(0.0, min(1.0, value))
    return value * value * (3.0 - 2.0 * value)


def soft_alpha(distance: int, transparent_threshold: float, opaque_threshold: float) -> int:
    if distance <= transparent_threshold:
        return 0
    if distance >= opaque_threshold:
        return 255
    ratio = (float(distance) - transparent_threshold) / (
        opaque_threshold - transparent_threshold
    )
    return clamp_channel(255.0 * smoothstep(ratio))


def spill_channels(key: Color) -> list[int]:
    key_max = max(key)
    if key_max < 128:
        return []
    return [idx for idx, value in enumerate(key) if value >= key_max - 16 and value >= 128]


def key_channel_dominance(rgb: Color, key: Color) -> float:
    spill = spill_channels(key)
    if not spill:
        return 0.0
    channels = [float(value) for value in rgb]
    non_spill = [idx for idx in range(3) if idx not in spill]
    key_strength = min(channels[idx] for idx in spill) if len(spill) > 1 else channels[spill[0]]
    non_key_strength = max((channels[idx] for idx in non_spill), default=0.0)
    return key_strength - non_key_strength


def dominance_alpha(rgb: Color, key: Color) -> int:
    spill = spill_channels(key)
    if not spill:
        return 255
    channels = [float(value) for value in rgb]
    non_spill = [idx for idx in range(3) if idx not in spill]
    key_strength = min(channels[idx] for idx in spill) if len(spill) > 1 else channels[spill[0]]
    non_key_strength = max((channels[idx] for idx in non_spill), default=0.0)
    dominance = key_strength - non_key_strength
    if dominance <= 0:
        return 255
    denominator = max(1.0, float(max(key)) - non_key_strength)
    alpha = 1.0 - min(1.0, dominance / denominator)
    return clamp_channel(alpha * 255.0)


def looks_key_colored(rgb: Color, key: Color, distance: int) -> bool:
    if distance <= 32:
        return True
    if not spill_channels(key):
        return True
    return key_channel_dominance(rgb, key) >= KEY_DOMINANCE_THRESHOLD


def cleanup_spill(rgb: Color, key: Color, alpha: int) -> Color:
    if alpha >= 252:
        return rgb
    spill = spill_channels(key)
    if not spill:
        return rgb
    channels = [float(value) for value in rgb]
    non_spill = [idx for idx in range(3) if idx not in spill]
    if non_spill:
        cap = max(0.0, max(channels[idx] for idx in non_spill) - 1.0)
        for idx in spill:
            channels[idx] = min(channels[idx], cap)
    return (clamp_channel(channels[0]), clamp_channel(channels[1]), clamp_channel(channels[2]))


def sample_key(image, mode: str) -> Color:
    pixels = image.load()
    width, height = image.size
    samples: list[Color] = []

    if mode == "corners":
        patch = max(1, min(width, height, 12))
        boxes = [
            (0, 0, patch, patch),
            (width - patch, 0, width, patch),
            (0, height - patch, patch, height),
            (width - patch, height - patch, width, height),
        ]
        for left, top, right, bottom in boxes:
            for y in range(top, bottom):
                for x in range(left, right):
                    samples.append(pixels[x, y][:3])
    elif mode == "border":
        band = max(1, min(width, height, 6))
        step = max(1, min(width, height) // 256)
        for x in range(0, width, step):
            for y in range(band):
                samples.append(pixels[x, y][:3])
                samples.append(pixels[x, height - 1 - y][:3])
        for y in range(0, height, step):
            for x in range(band):
                samples.append(pixels[x, y][:3])
                samples.append(pixels[width - 1 - x, y][:3])
    else:
        die(f"Unsupported auto-key mode: {mode}")

    if not samples:
        die("Could not sample key color.")
    return (
        int(round(median(sample[0] for sample in samples))),
        int(round(median(sample[1] for sample in samples))),
        int(round(median(sample[2] for sample in samples))),
    )


def remove_key(image, args: argparse.Namespace, key: Color) -> int:
    pixels = image.load()
    width, height = image.size
    transparent = 0

    for y in range(height):
        for x in range(width):
            red, green, blue, alpha = pixels[x, y]
            rgb = (red, green, blue)
            distance = channel_distance(rgb, key)
            key_like = looks_key_colored(rgb, key, distance)
            if args.soft_matte and key_like:
                output_alpha = min(
                    soft_alpha(distance, args.transparent_threshold, args.opaque_threshold),
                    dominance_alpha(rgb, key),
                )
            else:
                output_alpha = 0 if distance <= args.tolerance else 255

            output_alpha = int(round(output_alpha * (alpha / 255.0)))
            if 0 < output_alpha <= ALPHA_NOISE_FLOOR:
                output_alpha = 0

            if output_alpha == 0:
                pixels[x, y] = (0, 0, 0, 0)
                transparent += 1
            else:
                if args.despill and key_like:
                    red, green, blue = cleanup_spill(rgb, key, output_alpha)
                pixels[x, y] = (red, green, blue, output_alpha)

    return transparent


def remove_connected_background(image, args: argparse.Namespace, key: Color) -> int:
    pixels = image.load()
    width, height = image.size
    tolerance = args.connected_tolerance if args.connected_tolerance is not None else args.tolerance
    mask = connected_background_mask(image, key, tolerance)
    matched = 0

    for y in range(height):
        for x in range(width):
            if not mask[y * width + x]:
                continue
            red, green, blue, alpha = pixels[x, y]
            if alpha != 0:
                pixels[x, y] = (0, 0, 0, 0)
                matched += 1

    return matched


def connected_background_mask(image, key: Color, tolerance: int) -> bytearray:
    pixels = image.load()
    width, height = image.size
    visited = bytearray(width * height)
    mask = bytearray(width * height)
    queue: deque[tuple[int, int]] = deque()

    def index(x: int, y: int) -> int:
        return y * width + x

    def is_candidate(x: int, y: int) -> bool:
        red, green, blue, alpha = pixels[x, y]
        if alpha == 0:
            return True
        return channel_distance((red, green, blue), key) <= tolerance

    def push_if_candidate(x: int, y: int) -> None:
        idx = index(x, y)
        if visited[idx]:
            return
        visited[idx] = 1
        if is_candidate(x, y):
            queue.append((x, y))

    for x in range(width):
        push_if_candidate(x, 0)
        push_if_candidate(x, height - 1)
    for y in range(1, height - 1):
        push_if_candidate(0, y)
        push_if_candidate(width - 1, y)

    while queue:
        x, y = queue.popleft()
        mask[index(x, y)] = 1
        if x > 0:
            push_if_candidate(x - 1, y)
        if x < width - 1:
            push_if_candidate(x + 1, y)
        if y > 0:
            push_if_candidate(x, y - 1)
        if y < height - 1:
            push_if_candidate(x, y + 1)

    return mask


def flatten_connected_background(image, key: Color, tolerance: int):
    flattened = image.copy()
    pixels = flattened.load()
    width, height = flattened.size
    mask = connected_background_mask(flattened, key, tolerance)
    matched = 0

    for y in range(height):
        for x in range(width):
            if not mask[y * width + x]:
                continue
            _, _, _, alpha = pixels[x, y]
            pixels[x, y] = (key[0], key[1], key[2], alpha)
            matched += 1

    return flattened, matched


def flatten_key_like_pixels(image, key: Color, tolerance: int) -> int:
    pixels = image.load()
    width, height = image.size
    matched = 0

    for y in range(height):
        for x in range(width):
            red, green, blue, alpha = pixels[x, y]
            rgb = (red, green, blue)
            if rgb == key or channel_distance(rgb, key) > tolerance:
                continue
            pixels[x, y] = (key[0], key[1], key[2], alpha)
            matched += 1

    return matched


def contract_alpha(image, pixels: int):
    if pixels <= 0:
        return image
    _, ImageFilter, _ = load_pillow()
    alpha = image.getchannel("A")
    for _ in range(pixels):
        alpha = alpha.filter(ImageFilter.MinFilter(3))
    image.putalpha(alpha)
    return image


def feather_alpha(image, radius: float):
    if radius <= 0:
        return image
    _, ImageFilter, _ = load_pillow()
    alpha = image.getchannel("A").filter(ImageFilter.GaussianBlur(radius=radius))
    image.putalpha(alpha)
    return image


def trim_alpha(image, pad: int):
    _, _, ImageOps = load_pillow()
    alpha = image.getchannel("A")
    bbox = alpha.getbbox()
    if bbox is None:
        die("Cannot trim: output is fully transparent.")
    image = image.crop(bbox)
    if pad > 0:
        image = ImageOps.expand(image, border=pad, fill=(0, 0, 0, 0))
    return image


def resize_image(image, size: tuple[int, int], resample: str):
    Image, _, _ = load_pillow()
    filter_value = Image.Resampling.NEAREST if resample == "nearest" else Image.Resampling.LANCZOS
    return image.resize(size, filter_value)


def alpha_stats(image) -> tuple[int, int, int]:
    alpha = image.getchannel("A")
    if hasattr(alpha, "get_flattened_data"):
        values = list(alpha.get_flattened_data())
    else:
        values = list(alpha.getdata())
    transparent = sum(1 for value in values if value == 0)
    partial = sum(1 for value in values if 0 < value < 255)
    visible = sum(1 for value in values if value > 0)
    return transparent, partial, visible


def validate_output(image) -> None:
    if image.mode != "RGBA":
        die("Validation failed: output is not RGBA.")

    width, height = image.size
    pixels = image.load()
    corners = [pixels[0, 0][3], pixels[width - 1, 0][3], pixels[0, height - 1][3], pixels[width - 1, height - 1][3]]
    if any(alpha != 0 for alpha in corners):
        die("Validation failed: one or more corners are not fully transparent.")

    _, _, visible = alpha_stats(image)
    total = width * height
    if visible < max(4, total // 200):
        die("Validation failed: output has too little visible subject area.")


def validate_args(args: argparse.Namespace) -> None:
    src = Path(args.input)
    if not src.exists():
        die(f"Input image not found: {src}")
    out = Path(args.out)
    if out.exists() and not args.force:
        die(f"Output already exists: {out} (use --force to overwrite).")
    if out.suffix.lower() not in {".png", ".webp"}:
        die("--out must end in .png or .webp.")
    if args.flatten_background_out is not None:
        flat_out = Path(args.flatten_background_out)
        if flat_out.exists() and not args.force:
            die(f"Flattened output already exists: {flat_out} (use --force to overwrite).")
        if flat_out.suffix.lower() not in {".png", ".webp"}:
            die("--flatten-background-out must end in .png or .webp.")
    if args.tolerance < 0 or args.tolerance > 255:
        die("--tolerance must be between 0 and 255.")
    if args.transparent_threshold < 0 or args.transparent_threshold > 255:
        die("--transparent-threshold must be between 0 and 255.")
    if args.opaque_threshold < 0 or args.opaque_threshold > 255:
        die("--opaque-threshold must be between 0 and 255.")
    if args.soft_matte and args.transparent_threshold >= args.opaque_threshold:
        die("--transparent-threshold must be lower than --opaque-threshold.")
    if args.edge_contract < 0 or args.edge_contract > 16:
        die("--edge-contract must be between 0 and 16.")
    if args.edge_feather < 0 or args.edge_feather > 64:
        die("--edge-feather must be between 0 and 64.")
    if args.pad < 0 or args.pad > 512:
        die("--pad must be between 0 and 512.")
    if args.connected_tolerance is not None and (
        args.connected_tolerance < 0 or args.connected_tolerance > 255
    ):
        die("--connected-tolerance must be between 0 and 255.")
    if args.flatten_disconnected_threshold is not None and (
        args.flatten_disconnected_threshold < 0 or args.flatten_disconnected_threshold > 255
    ):
        die("--flatten-disconnected-threshold must be between 0 and 255.")


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Remove a flat chroma-key background.")
    parser.add_argument("--input", required=True, help="Input image path.")
    parser.add_argument("--out", required=True, help="Output .png or .webp path.")
    parser.add_argument("--key-color", default="#00ff00", help="Explicit key color, e.g. #00ff00.")
    parser.add_argument("--auto-key", choices=["none", "corners", "border"], default="none", help="Sample key color instead of using --key-color.")
    parser.add_argument("--tolerance", type=int, default=12, help="Hard key per-channel tolerance.")
    parser.add_argument("--soft-matte", action="store_true", help="Use a smooth alpha ramp at edges.")
    parser.add_argument("--transparent-threshold", type=float, default=12.0, help="Soft matte transparent threshold.")
    parser.add_argument("--opaque-threshold", type=float, default=96.0, help="Soft matte opaque threshold.")
    parser.add_argument("--edge-contract", type=int, default=0, help="Shrink alpha edge by N pixels.")
    parser.add_argument("--edge-feather", type=float, default=0.0, help="Feather alpha edge by radius.")
    parser.add_argument("--despill", action="store_true", help="Reduce key-color spill on edge pixels.")
    parser.add_argument("--connected-background", action="store_true", help="Remove only key-like pixels connected to the image border.")
    parser.add_argument("--connected-tolerance", type=int, help="Tolerance used for --connected-background; defaults to --tolerance.")
    parser.add_argument("--flatten-background-out", help="Optional path for a copy with connected background pixels flattened to the key color.")
    parser.add_argument("--flatten-disconnected-threshold", type=int, help="When flattening, also flatten disconnected pixels within this distance of the key color.")
    parser.add_argument("--trim", action="store_true", help="Crop to alpha bounding box.")
    parser.add_argument("--pad", type=int, default=0, help="Transparent padding after --trim.")
    parser.add_argument("--resize", help="Optional final output size, e.g. 64x64.")
    parser.add_argument("--resample", choices=["nearest", "lanczos"], default="nearest", help="Resize filter.")
    parser.add_argument("--validate", action="store_true", help="Validate alpha output.")
    parser.add_argument("--force", action="store_true", help="Overwrite output.")
    return parser


def main() -> None:
    parser = build_parser()
    args = parser.parse_args()
    validate_args(args)
    resize = parse_size(args.resize)
    Image, _, _ = load_pillow()

    with Image.open(args.input) as source:
        image = source.convert("RGBA")

    key = sample_key(image, args.auto_key) if args.auto_key != "none" else parse_key_color(args.key_color)
    if args.flatten_background_out is not None:
        tolerance = args.connected_tolerance if args.connected_tolerance is not None else args.tolerance
        flattened, flat_matched = flatten_connected_background(image, key, tolerance)
        disconnected_matched = (
            flatten_key_like_pixels(flattened, key, args.flatten_disconnected_threshold)
            if args.flatten_disconnected_threshold is not None
            else 0
        )
        flat_out = Path(args.flatten_background_out)
        flat_out.parent.mkdir(parents=True, exist_ok=True)
        flattened.save(flat_out)
        print(f"Wrote flattened background source {flat_out}")
        print(f"Flattened connected pixels: {flat_matched}")
        if args.flatten_disconnected_threshold is not None:
            print(f"Flattened disconnected key-like pixels: {disconnected_matched}")
    matched = (
        remove_connected_background(image, args, key)
        if args.connected_background
        else remove_key(image, args, key)
    )
    image = contract_alpha(image, args.edge_contract)
    image = feather_alpha(image, args.edge_feather)
    if args.trim:
        image = trim_alpha(image, args.pad)
    if resize is not None:
        image = resize_image(image, resize, args.resample)
    if args.validate:
        validate_output(image)

    out = Path(args.out)
    out.parent.mkdir(parents=True, exist_ok=True)
    image.save(out)
    transparent, partial, visible = alpha_stats(image)
    total = image.size[0] * image.size[1]
    print(f"Wrote {out}")
    print(f"Key color: #{key[0]:02x}{key[1]:02x}{key[2]:02x}")
    print(f"Matched source pixels: {matched}")
    print(f"Transparent pixels: {transparent}/{total}")
    print(f"Partially transparent pixels: {partial}/{total}")
    print(f"Visible pixels: {visible}/{total}")
    if matched == 0:
        print("Warning: no source pixels matched the key color.", file=sys.stderr)


if __name__ == "__main__":
    main()
