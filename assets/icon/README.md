# Ikona aplikace

- `icon.svg` → `icon.png` (1024 × 1024, bez alfy): plná ikona, pozadí + puk + vlna.
- `icon_foreground.svg` → `icon_foreground.png` (1024 × 1024, s alfou): popředí adaptivní ikony Androidu. Generátor (`flutter_launcher_icons`) přidá inset 16 % sám, popředí se dál nezmenšuje.
- Pozadí adaptivní ikony: `#173a5c` (horní barva přechodu).

Vlna nad pukem je „vlna“ týmu z plánu (forma), ne graf zisku. Obsah je zmenšený na 82 %, aby se vešel do kulaté masky.

Render: SVG přes Quick Look (`qlmanage -t -s 1024`); průhledné popředí z rozdílu renderu na černém a bílém pozadí (ImageMagick).
