#!/usr/bin/env python3
"""Generate the multi-resolution app icon and README PNG from one drawing."""
from __future__ import annotations

from pathlib import Path
from PIL import Image, ImageDraw

ROOT = Path(__file__).resolve().parent.parent
ASSETS = ROOT / "assets"
DOCS = ROOT / "docs"
APP_ASSETS = ROOT / "internal" / "assets"
SETTINGS_ASSETS = ROOT / "internal" / "settings" / "assets"
ASSETS.mkdir(exist_ok=True)
DOCS.mkdir(exist_ok=True)
APP_ASSETS.mkdir(exist_ok=True)
SETTINGS_ASSETS.mkdir(exist_ok=True)

S = 4096
img = Image.new("RGBA", (S, S), (0, 0, 0, 0))
d = ImageDraw.Draw(img)

# Vertical gradient rounded square.
top = (37, 99, 235)
bottom = (6, 182, 212)
margin = int(S * 0.055)
radius = int(S * 0.225)
for y in range(margin, S - margin):
    t = (y - margin) / max(1, S - 2 * margin)
    color = tuple(int(top[i] * (1 - t) + bottom[i] * t) for i in range(3)) + (255,)
    d.line([(margin, y), (S - margin, y)], fill=color)
mask = Image.new("L", (S, S), 0)
ImageDraw.Draw(mask).rounded_rectangle(
    [margin, margin, S - margin, S - margin], radius=radius, fill=255
)
rounded = Image.new("RGBA", (S, S), (0, 0, 0, 0))
rounded.paste(img, (0, 0), mask)
img = rounded
d = ImageDraw.Draw(img)

# Subtle top highlight.
glow = Image.new("RGBA", (S, S), (0, 0, 0, 0))
gd = ImageDraw.Draw(glow)
gd.ellipse([-S * 0.25, -S * 0.55, S * 0.9, S * 0.35], fill=(255, 255, 255, 36))
img.alpha_composite(glow)
d = ImageDraw.Draw(img)

# Document sheet.
doc = [int(S * 0.255), int(S * 0.185), int(S * 0.705), int(S * 0.79)]
doc_radius = int(S * 0.045)
d.rounded_rectangle(doc, radius=doc_radius, fill=(255, 255, 255, 246))
# Folded corner.
fold = int(S * 0.115)
x2, y1 = doc[2], doc[1]
d.polygon(
    [(x2 - fold, y1), (x2, y1), (x2, y1 + fold)],
    fill=(186, 230, 253, 255),
)
d.line([(x2 - fold, y1), (x2 - fold, y1 + fold), (x2, y1 + fold)],
       fill=(125, 211, 252, 255), width=int(S * 0.008))

# Text lines: one selected range band and two regular lines.
line_x1 = int(S * 0.34)
line_x2 = int(S * 0.60)
band_top = int(S * 0.365)
band_h = int(S * 0.085)
d.rounded_rectangle(
    [int(S * 0.305), band_top, int(S * 0.655), band_top + band_h],
    radius=int(S * 0.018),
    fill=(14, 165, 233, 255),
)
d.rounded_rectangle(
    [line_x1, band_top + int(S * 0.025), line_x2, band_top + int(S * 0.058)],
    radius=int(S * 0.012),
    fill=(255, 255, 255, 255),
)
for i, width_frac in enumerate((0.26, 0.20)):
    y = int(S * (0.52 + i * 0.09))
    d.rounded_rectangle(
        [line_x1, y, line_x1 + int(S * width_frac), y + int(S * 0.035)],
        radius=int(S * 0.012),
        fill=(30, 64, 175, 235),
    )

# Location pin overlapping the lower-right corner.
pin_cx, pin_cy, pin_r = int(S * 0.715), int(S * 0.72), int(S * 0.145)
d.ellipse(
    [pin_cx - pin_r, pin_cy - pin_r, pin_cx + pin_r, pin_cy + pin_r],
    fill=(251, 113, 133, 255),
)
d.polygon(
    [
        (pin_cx - int(pin_r * 0.60), pin_cy + int(pin_r * 0.45)),
        (pin_cx + int(pin_r * 0.60), pin_cy + int(pin_r * 0.45)),
        (pin_cx, pin_cy + int(pin_r * 1.30)),
    ],
    fill=(251, 113, 133, 255),
)
d.ellipse(
    [pin_cx - int(pin_r * 0.42), pin_cy - int(pin_r * 0.42),
     pin_cx + int(pin_r * 0.42), pin_cy + int(pin_r * 0.42)],
    fill=(255, 255, 255, 255),
)
d.ellipse(
    [pin_cx - int(pin_r * 0.20), pin_cy - int(pin_r * 0.20),
     pin_cx + int(pin_r * 0.20), pin_cy + int(pin_r * 0.20)],
    fill=(190, 24, 93, 255),
)

# Trim transparent padding to a clean square.
bbox = img.getbbox()
if bbox:
    img = img.crop(bbox)
img = img.resize((1024, 1024), Image.LANCZOS)

png512 = img.resize((512, 512), Image.LANCZOS)
png512.save(DOCS / "icon.png")
png512.save(ASSETS / "icon.png")
png512.save(SETTINGS_ASSETS / "icon.png")

sizes = [16, 20, 24, 32, 40, 48, 64, 128, 256]
frames = [img.resize((n, n), Image.LANCZOS) for n in sizes]
for target in (ASSETS / "icon.ico", APP_ASSETS / "icon.ico"):
    frames[-1].save(
        target,
        format="ICO",
        sizes=[(n, n) for n in sizes],
        append_images=frames[:-1],
    )
print("wrote", ASSETS / "icon.ico", APP_ASSETS / "icon.ico", ASSETS / "icon.png", DOCS / "icon.png")
