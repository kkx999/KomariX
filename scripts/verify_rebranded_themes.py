#!/usr/bin/env python3
import argparse
import hashlib
import json
import re
import zipfile
from pathlib import Path

TEXT_EXTS = {
    ".html", ".htm", ".js", ".mjs", ".cjs", ".css", ".json", ".txt",
    ".md", ".svg", ".xml", ".webmanifest", ".ts", ".tsx", ".jsx", ".vue",
}
LEGAL_NAMES = {"LICENSE", "LICENSE.md", "LICENSE.txt", "NOTICE", "COPYING", "COPYING.md", "AUTHORS", "AUTHORS.md"}
URL_RX = re.compile(r"https?://[^\s\"'<>)]+")

FORBIDDEN = [
    ("powered", re.compile(r"Powered\s+by\s+Komari(?!X)", re.I)),
    ("monitor", re.compile(r"\bKomari\s+Monitor\b", re.I)),
    ("github", re.compile(r"https?://(?:www\.)?github\.com/komari-monitor/komari(?:[^\s\"'<>)]*)?", re.I)),
    ("raw", re.compile(r"https?://raw\.githubusercontent\.com/komari-monitor/komari(?:[^\s\"'<>)]*)?", re.I)),
    ("word", re.compile(r"\bKomari\b")),
    ("upper", re.compile(r"\bKOMARI\b")),
]

def sha256(path: Path) -> str:
    h = hashlib.sha256()
    with path.open("rb") as f:
        for chunk in iter(lambda: f.read(1024 * 1024), b""):
            h.update(chunk)
    return h.hexdigest()

def is_text(name: str) -> bool:
    p = Path(name)
    if p.name in LEGAL_NAMES:
        return False
    return p.suffix.lower() in TEXT_EXTS or p.name == "komari-theme.json"

def strip_ignored(text: str) -> str:
    # Author/source URLs may legitimately contain "komari" in the third-party
    # repository name. They are attribution, not product branding.
    text = URL_RX.sub("__URL__", text)

    # Keep original upstream copyright wording intact when embedded in bundles.
    text = re.sub(
        r"Copyright\s*(?:\([Cc]\)|©)?[^\n\"']{0,160}?\bKomari(?:\s+Monitor)?\b",
        "__LEGAL_COPYRIGHT__",
        text,
        flags=re.I,
    )
    return text

def get_manifest(z: zipfile.ZipFile):
    candidates = [n for n in z.namelist() if Path(n).name == "komari-theme.json"]
    if len(candidates) != 1:
        raise RuntimeError(f"expected one komari-theme.json, got {candidates}")
    return json.loads(z.read(candidates[0]).decode("utf-8"))

def legal_payloads(z: zipfile.ZipFile):
    out = {}
    for n in z.namelist():
        if Path(n).name in LEGAL_NAMES and not n.endswith("/"):
            out[n] = z.read(n)
    return out

def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--source-assets", required=True)
    ap.add_argument("--rebranded-assets", required=True)
    ap.add_argument("--source-catalog", required=True)
    ap.add_argument("--rebranded-catalog", required=True)
    args = ap.parse_args()

    source_assets = Path(args.source_assets)
    out_assets = Path(args.rebranded_assets)
    source_catalog = json.loads(Path(args.source_catalog).read_text(encoding="utf-8"))
    out_catalog = json.loads(Path(args.rebranded_catalog).read_text(encoding="utf-8"))
    source_themes = source_catalog if isinstance(source_catalog, list) else source_catalog["themes"]
    out_themes = out_catalog if isinstance(out_catalog, list) else out_catalog["themes"]

    if len(source_themes) != 49 or len(out_themes) != 49:
        raise RuntimeError(f"expected 49 catalog entries, got source={len(source_themes)} output={len(out_themes)}")

    out_zips = sorted(out_assets.glob("theme-*.zip"))
    if len(out_zips) != 49:
        raise RuntimeError(f"expected 49 rebranded ZIPs, got {len(out_zips)}")

    failures = []
    for idx, (src_meta, out_meta) in enumerate(zip(source_themes, out_themes), start=1):
        src_candidates = sorted(source_assets.glob(f"theme-{idx:02d}-*.zip"))
        out_candidates = sorted(out_assets.glob(f"theme-{idx:02d}-*.zip"))
        if len(src_candidates) != 1 or len(out_candidates) != 1:
            failures.append(f"#{idx}: package lookup failed")
            continue

        src_zip = src_candidates[0]
        out_zip = out_candidates[0]

        actual = sha256(out_zip)
        if actual.lower() != str(out_meta.get("sha256", "")).lower():
            failures.append(f"#{idx} {out_meta.get('short')}: SHA256 mismatch")

        if out_meta.get("short") != src_meta.get("short"):
            failures.append(f"#{idx}: short changed")
        if out_meta.get("author") != src_meta.get("author"):
            failures.append(f"#{idx}: author changed")
        if out_meta.get("version") != src_meta.get("version"):
            failures.append(f"#{idx}: version changed")

        with zipfile.ZipFile(src_zip) as zin, zipfile.ZipFile(out_zip) as zout:
            try:
                src_manifest = get_manifest(zin)
                out_manifest = get_manifest(zout)
            except Exception as exc:
                failures.append(f"#{idx}: manifest error: {exc}")
                continue

            if src_manifest.get("short") != out_manifest.get("short"):
                failures.append(f"#{idx}: manifest short changed")
            if src_manifest.get("author") != out_manifest.get("author"):
                failures.append(f"#{idx}: manifest author changed")

            src_legal = legal_payloads(zin)
            out_legal = legal_payloads(zout)
            if src_legal != out_legal:
                failures.append(f"#{idx}: LICENSE/NOTICE/AUTHORS payload changed")

            for info in zout.infolist():
                if info.is_dir() or not is_text(info.filename):
                    continue
                try:
                    text = zout.read(info).decode("utf-8")
                except Exception:
                    continue

                if Path(info.filename).name == "komari-theme.json":
                    try:
                        parsed = json.loads(text)
                        ignored_keys = {"short", "author", "authors", "url", "homepage", "repository", "source", "preview"}
                        visible = []
                        def collect(obj, key=None):
                            if isinstance(obj, dict):
                                for k, v in obj.items():
                                    collect(v, k)
                            elif isinstance(obj, list):
                                for v in obj:
                                    collect(v, key)
                            elif isinstance(obj, str) and key not in ignored_keys:
                                visible.append(obj)
                        collect(parsed)
                        audit_text = "\n".join(visible)
                    except Exception:
                        audit_text = text
                else:
                    audit_text = strip_ignored(text)

                for kind, rx in FORBIDDEN:
                    m = rx.search(audit_text)
                    if m:
                        sample = audit_text[max(0,m.start()-80):min(len(audit_text),m.end()+120)]
                        failures.append(
                            f"#{idx} {out_meta.get('short')} {info.filename} [{kind}] {sample.replace(chr(10),' ')[:300]}"
                        )
                        break

        print(f"[OK] {idx:02d}/49 {out_meta.get('short')}", flush=True)

    if failures:
        print("\nVerification failures:")
        for item in failures[:200]:
            print(" -", item)
        raise SystemExit(1)

    print("Verified 49/49 rebranded themes: branding clean, SHA valid, authors and legal files preserved.")

if __name__ == "__main__":
    main()
