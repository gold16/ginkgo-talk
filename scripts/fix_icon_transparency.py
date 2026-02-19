"""
Fix icon transparency: removes white/near-white fringe from PNG source
and generates a clean .ico file with multiple sizes.
Uses only Pillow (no numpy required).
"""
import sys
from pathlib import Path
from PIL import Image


def remove_white_fringe(img: Image.Image, threshold: int = 240) -> Image.Image:
    """
    Remove white/near-white background and fringe from an RGBA image.
    Pixels where R, G, B are all >= threshold become fully transparent.
    Semi-transparent edge pixels have their white blend reversed.
    """
    img = img.convert("RGBA")
    pixels = img.load()
    w, h = img.size

    for y in range(h):
        for x in range(w):
            r, g, b, a = pixels[x, y]

            # Fully white/near-white -> transparent
            if r >= threshold and g >= threshold and b >= threshold:
                pixels[x, y] = (0, 0, 0, 0)
                continue

            # Semi-transparent pixels: reverse white premultiply
            if 0 < a < 255:
                a_norm = a / 255.0
                nr = int(max(0, min(255, (r - 255 * (1 - a_norm)) / a_norm)))
                ng = int(max(0, min(255, (g - 255 * (1 - a_norm)) / a_norm)))
                nb = int(max(0, min(255, (b - 255 * (1 - a_norm)) / a_norm)))
                pixels[x, y] = (nr, ng, nb, a)

    return img


def generate_ico(source_path: str, out_ico: str, sizes=(256, 48, 32, 16)):
    """Generate a multi-size .ico file from a source PNG."""
    src = Image.open(source_path).convert("RGBA")

    # Crop to square (center crop)
    w, h = src.size
    side = min(w, h)
    left = (w - side) // 2
    top = (h - side) // 2
    src = src.crop((left, top, left + side, top + side))

    # Remove white fringe
    print("Removing white fringe (this may take a moment)...")
    src = remove_white_fringe(src)

    # Save cleaned source for verification
    clean_src = str(Path(source_path).parent / "icon_cleaned.png")
    clean_512 = src.resize((512, 512), Image.LANCZOS)
    clean_512.save(clean_src, format="PNG")
    print(f"Saved cleaned source: {clean_src}")

    # Generate each size with high-quality resampling
    icons = []
    for size in sizes:
        resized = src.resize((size, size), Image.LANCZOS)
        icons.append(resized)
        print(f"  Generated {size}x{size}")

    # Save as .ico
    icons[0].save(out_ico, format="ICO", sizes=[(s, s) for s in sizes],
                  append_images=icons[1:])
    print(f"Saved: {out_ico}")

    # Also save web icons
    web_dir = Path(source_path).parent / "web"
    for sz in (512, 192, 32):
        clean = src.resize((sz, sz), Image.LANCZOS)
        p = str(web_dir / f"icon-{sz}.png")
        clean.save(p, format="PNG")
        print(f"Saved: {p}")

    # Generate tray icon too
    tray = src.resize((32, 32), Image.LANCZOS)
    tray_path = str(web_dir / "tray-icon.ico")
    tray.save(tray_path, format="ICO", sizes=[(32, 32)])
    print(f"Saved: {tray_path}")


if __name__ == "__main__":
    source = sys.argv[1] if len(sys.argv) > 1 else "ginkgo_purple_1771297597058_transparent.png"
    ico_out = sys.argv[2] if len(sys.argv) > 2 else "app.ico"
    print(f"Source: {source}")
    print(f"Output: {ico_out}")
    generate_ico(source, ico_out)
    print("Done!")
