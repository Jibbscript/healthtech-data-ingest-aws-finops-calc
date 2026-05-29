#!/usr/bin/env python3
"""Render docs/one-pager.md into docs/one-pager.pdf.

The one-pager PDF is a committed, visitor-facing deliverable (an exec summary
suitable for a pre-read). It is generated from docs/one-pager.md so the two can
never drift -- re-run this script (or `make one-pager`) whenever the markdown
changes.

Requires reportlab (`pip install reportlab`); ReportLab is the same producer the
original PDF was built with, so the output stays consistent.
"""
from __future__ import annotations

import html
import re
import sys
from pathlib import Path

try:
    from reportlab import rl_config
    from reportlab.lib.enums import TA_LEFT
    from reportlab.lib.pagesizes import LETTER
    from reportlab.lib.styles import ParagraphStyle
    from reportlab.lib.units import inch
    from reportlab.platypus import Paragraph, SimpleDocTemplate, Spacer
except ImportError:  # pragma: no cover - dependency hint
    sys.exit("reportlab is required to build the one-pager PDF: pip install reportlab")

# Deterministic output: fixed timestamp + content-derived document ID instead of
# wall-clock dates and a random ID. This makes regeneration idempotent, so an
# unchanged docs/one-pager.md yields a byte-identical PDF (no spurious git diffs).
rl_config.invariant = 1

REPO_ROOT = Path(__file__).resolve().parent.parent
SOURCE = REPO_ROOT / "docs" / "one-pager.md"
OUTPUT = REPO_ROOT / "docs" / "one-pager.pdf"

# The standard PDF fonts (Helvetica) use WinAnsi encoding, which has no glyph
# for some Unicode punctuation; those would render as missing-glyph boxes. Map
# them to ASCII equivalents. The original one-pager already used "->" for arrows.
_GLYPH_FALLBACKS = {
    "→": "->",  # rightwards arrow
    "–": "-",  # en dash
    "—": "--",  # em dash
    "‘": "'",
    "’": "'",
    "“": '"',
    "”": '"',
}


def _to_markup(text: str, where: str = "text") -> str:
    """Convert a single markdown paragraph into ReportLab inline markup."""
    for unicode_char, ascii_char in _GLYPH_FALLBACKS.items():
        text = text.replace(unicode_char, ascii_char)
    # Helvetica only renders WinAnsi (cp1252). Fail loudly on any other glyph
    # rather than letting reportlab emit a silent garbage box; the fix is to add
    # an ASCII fallback above.
    try:
        text.encode("cp1252")
    except UnicodeEncodeError as exc:
        bad = text[exc.start : exc.end]
        sys.exit(
            f"{where}: unsupported glyph {bad!r} (U+{ord(bad[0]):04X}); "
            "add an ASCII fallback to _GLYPH_FALLBACKS in this script"
        )
    text = html.escape(text, quote=False)  # neutralize &, <, > before adding tags
    text = re.sub(r"\*\*(.+?)\*\*", r"<b>\1</b>", text)  # **bold** -> <b>bold</b>
    return text


def _parse(markdown: str) -> tuple[str, list[str]]:
    """Split the one-pager into its title and body paragraphs."""
    title: str | None = None
    paragraphs: list[str] = []
    for raw_line in markdown.splitlines():
        line = raw_line.strip()
        if not line:
            continue
        if title is None and line.startswith("# "):
            title = line[2:].strip()
        else:
            paragraphs.append(line)
    return title or "One-Pager", paragraphs


def main() -> int:
    if not SOURCE.exists():
        sys.exit(f"source markdown not found: {SOURCE}")
    title, paragraphs = _parse(SOURCE.read_text(encoding="utf-8"))

    title_style = ParagraphStyle(
        "OnePagerTitle", fontName="Helvetica-Bold", fontSize=18, leading=22, spaceAfter=14
    )
    body_style = ParagraphStyle(
        "OnePagerBody", fontName="Helvetica", fontSize=10.5, leading=15, spaceAfter=10, alignment=TA_LEFT
    )

    flowables = [Paragraph(_to_markup(title, "title"), title_style), Spacer(1, 4)]
    flowables += [
        Paragraph(_to_markup(p, f"paragraph {i}"), body_style)
        for i, p in enumerate(paragraphs, start=1)
    ]

    doc = SimpleDocTemplate(
        str(OUTPUT),
        pagesize=LETTER,
        leftMargin=0.9 * inch,
        rightMargin=0.9 * inch,
        topMargin=0.9 * inch,
        bottomMargin=0.9 * inch,
        title=title,
        author="Throne Ingest PoC",
        subject="Throne Ingest PoC one-pager",
    )
    doc.build(flowables)
    print(
        f"wrote {OUTPUT.relative_to(REPO_ROOT)} ({OUTPUT.stat().st_size} bytes) "
        f"from {SOURCE.relative_to(REPO_ROOT)} -- title + {len(paragraphs)} paragraphs"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
