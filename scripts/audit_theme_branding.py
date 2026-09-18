#!/usr/bin/env python3
import argparse
import json
import re
import zipfile
from pathlib import Path

TEXT_EXTS = {
    ".html", ".htm", ".js", ".mjs", ".cjs", ".css", ".json", ".txt",
    ".md", ".svg", ".xml", ".webmanifest", ".ts", ".tsx", ".jsx", ".vue",
}
SKIP_NAMES = {"LICENSE", "LICENSE.md", "LICENSE.txt", "NOTICE", "COPYING", "COPYING.md"}

PATTERNS = [
    ("powered", re.compile(r"Powered\s+by\s+Komari", re.I)),
    ("monitor", re.compile(r"Komari\s+Monitor", re.I)),
    ("github", re.compile(r"https?://(?:www\.)?github\.com/komari-monitor/komari(?:[^\s\"'<>)]*)?", re.I)),
    ("raw", re.compile(r"https?://raw\.githubusercontent\.com/komari-monitor/komari(?:[^\s\"'<>)]*)?", re.I)),
    ("visible_word", re.compile(r"\bKomari\b")),
]

def is_text(path: Path) -> bool:
    if path.name in SKIP_NAMES:
        return False
    return path.suffix.lower() in TEXT_EXTS or path.name in {"komarix-theme.json", "komari-theme.json"}

def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--assets", required=True)
    ap.add_argument("--output", required=True)
    args = ap.parse_args()

    assets = Path(args.assets)
    output = Path(args.output)
    output.parent.mkdir(parents=True, exist_ok=True)

    report = []
    for z in sorted(assets.glob("theme-*.zip")):
        theme_hits = []
        with zipfile.ZipFile(z) as archive:
            for info in archive.infolist():
                if info.is_dir():
                    continue
                p = Path(info.filename)
                if not is_text(p):
                    continue
                try:
                    text = archive.read(info).decode("utf-8")
                except Exception:
                    continue
                for kind, rx in PATTERNS:
                    matches = list(rx.finditer(text))
                    if not matches:
                        continue
                    samples = []
                    for m in matches[:5]:
                        start = max(0, m.start() - 80)
                        end = min(len(text), m.end() + 120)
                        sample = text[start:end].replace("\n", " ").replace("\r", " ")
                        samples.append(sample)
                    theme_hits.append({
                        "file": info.filename,
                        "kind": kind,
                        "count": len(matches),
                        "samples": samples,
                    })
        report.append({
            "package": z.name,
            "hit_files": len({h["file"] for h in theme_hits}),
            "hits": theme_hits,
        })

    output.write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")

    affected = [r for r in report if r["hits"]]
    print(f"Scanned {len(report)} theme packages; {len(affected)} contain legacy branding candidates.")
    for item in affected:
        kinds = {}
        for h in item["hits"]:
            kinds[h["kind"]] = kinds.get(h["kind"], 0) + h["count"]
        print(f"{item['package']}: files={item['hit_files']} hits={kinds}")

if __name__ == "__main__":
    main()
