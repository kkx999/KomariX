#!/usr/bin/env python3
import argparse
import hashlib
import json
import re
import shutil
import zipfile
from pathlib import Path

TEXT_EXTS = {
    ".html", ".htm", ".js", ".mjs", ".cjs", ".css", ".json", ".txt",
    ".md", ".svg", ".xml", ".webmanifest", ".ts", ".tsx", ".jsx", ".vue",
}
LEGAL_NAMES = {"LICENSE", "LICENSE.md", "LICENSE.txt", "NOTICE", "COPYING", "COPYING.md", "AUTHORS", "AUTHORS.md"}
URL_RX = re.compile(r"https?://[^\s\"'<>)]+")

CORE_URL_REPLACEMENTS = (
    ("https://github.com/komari-monitor/komari/blob/main/README.md", "https://github.com/kkx999/KomariX/blob/main/README.md"),
    ("https://github.com/komari-monitor/komari", "https://github.com/kkx999/KomariX"),
    ("https://raw.githubusercontent.com/komari-monitor/komari/refs/heads/main/README.md", "https://raw.githubusercontent.com/kkx999/KomariX/main/README.md"),
    ("https://raw.githubusercontent.com/komari-monitor/komari/main/README.md", "https://raw.githubusercontent.com/kkx999/KomariX/main/README.md"),
)

def is_text(name: str) -> bool:
    p = Path(name)
    if p.name in LEGAL_NAMES:
        return False
    return p.suffix.lower() in TEXT_EXTS or p.name in {"komarix-theme.json", "komari-theme.json"}

def protect_urls(text: str):
    saved = []
    def repl(m):
        url = m.group(0)
        for old, new in CORE_URL_REPLACEMENTS:
            if url.startswith(old):
                url = new + url[len(old):]
                break
        token = f"__KOMARIX_URL_{len(saved)}__"
        saved.append(url)
        return token
    return URL_RX.sub(repl, text), saved

def restore_urls(text: str, saved):
    for i, url in enumerate(saved):
        text = text.replace(f"__KOMARIX_URL_{i}__", url)
    return text

def protect_legal_brand(text: str):
    saved = []
    rx = re.compile(r"Copyright\s*(?:\([Cc]\)|©)?[^\n\"']{0,120}?\bKomari(?:\s+Monitor)?\b", re.I)
    def repl(m):
        token = f"__KOMARIX_LEGAL_{len(saved)}__"
        saved.append(m.group(0))
        return token
    return rx.sub(repl, text), saved

def restore_legal(text: str, saved):
    for i, value in enumerate(saved):
        text = text.replace(f"__KOMARIX_LEGAL_{i}__", value)
    return text

def replace_brand_text(text: str) -> str:
    text, urls = protect_urls(text)
    text, legal = protect_legal_brand(text)

    text = re.sub(r"\bKomari\s+Monitor\b", "KomariX Monitor", text)
    text = re.sub(r"\bKOMARI\s+MONITOR\b", "KOMARIX MONITOR", text)
    text = re.sub(r"\bKomari\b", "KomariX", text)
    text = re.sub(r"\bKOMARI\b", "KOMARIX", text)

    text = restore_legal(text, legal)
    text = restore_urls(text, urls)
    return text

def replace_json_values(obj, key=None):
    preserve_keys = {"short", "author", "authors"}
    url_keys = {"url", "homepage", "repository", "source", "preview"}

    if isinstance(obj, dict):
        return {k: replace_json_values(v, k) for k, v in obj.items()}
    if isinstance(obj, list):
        return [replace_json_values(v, key) for v in obj]
    if isinstance(obj, str):
        if key in preserve_keys:
            return obj
        if key in url_keys:
            for old, new in CORE_URL_REPLACEMENTS:
                if obj.startswith(old):
                    return new + obj[len(old):]
            return obj
        return replace_brand_text(obj)
    return obj

def transform_bytes(name: str, data: bytes) -> bytes:
    if not is_text(name):
        return data
    try:
        text = data.decode("utf-8")
    except UnicodeDecodeError:
        return data

    if Path(name).name in {"komarix-theme.json", "komari-theme.json"}:
        try:
            parsed = json.loads(text)
            parsed = replace_json_values(parsed)
            return (json.dumps(parsed, ensure_ascii=False, indent=2) + "\n").encode("utf-8")
        except Exception:
            pass

    return replace_brand_text(text).encode("utf-8")

def repack(src: Path, tmp_out: Path):
    with zipfile.ZipFile(src, "r") as zin, zipfile.ZipFile(tmp_out, "w") as zout:
        names = {Path(info.filename).name for info in zin.infolist() if not info.is_dir()}
        for info in zin.infolist():
            data = zin.read(info.filename) if not info.is_dir() else b""
            if not info.is_dir():
                data = transform_bytes(info.filename, data)
            output_name = info.filename
            if Path(info.filename).name == "komari-theme.json" and "komarix-theme.json" not in names:
                output_name = str(Path(info.filename).with_name("komarix-theme.json")).replace("\\", "/")
            new_info = zipfile.ZipInfo(output_name, date_time=info.date_time)
            new_info.compress_type = info.compress_type
            new_info.comment = info.comment
            new_info.extra = info.extra
            new_info.internal_attr = info.internal_attr
            new_info.external_attr = info.external_attr
            new_info.create_system = info.create_system
            new_info.flag_bits = info.flag_bits
            new_info.volume = info.volume
            zout.writestr(new_info, data)

def sha256(path: Path) -> str:
    h = hashlib.sha256()
    with path.open("rb") as f:
        for chunk in iter(lambda: f.read(1024 * 1024), b""):
            h.update(chunk)
    return h.hexdigest()

def slug(value) -> str:
    value = re.sub(r"[^A-Za-z0-9._-]+", "-", str(value or "").strip()).strip("-.")
    return value[:80] or "theme"

def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--assets", required=True)
    ap.add_argument("--catalog", required=True)
    ap.add_argument("--output-dir", required=True)
    ap.add_argument("--release-tag", required=True)
    args = ap.parse_args()

    assets = Path(args.assets)
    out = Path(args.output_dir)
    out.mkdir(parents=True, exist_ok=True)

    raw = json.loads(Path(args.catalog).read_text(encoding="utf-8"))
    catalog = json.loads(json.dumps(raw))
    themes = catalog if isinstance(catalog, list) else catalog["themes"]

    for preview in assets.glob("preview-*"):
        shutil.copy2(preview, out / preview.name)

    report = []
    for idx, theme in enumerate(themes, start=1):
        candidates = sorted(assets.glob(f"theme-{idx:02d}-*.zip"))
        if len(candidates) != 1:
            raise RuntimeError(f"theme #{idx}: expected exactly one source ZIP, got {len(candidates)}")
        src = candidates[0]

        tmp = out / f".theme-{idx:02d}.tmp.zip"
        repack(src, tmp)
        digest = sha256(tmp)

        short = slug(theme.get("short", f"theme-{idx}"))
        version = slug(theme.get("version", "unknown"))
        final_name = f"theme-{idx:02d}-{short}-{version}-{digest[:12]}.zip"
        final_path = out / final_name
        tmp.replace(final_path)

        theme["download"] = f"https://github.com/kkx999/KomariX/releases/download/{args.release_tag}/{final_name}"
        theme["sha256"] = digest
        theme["komarix_branding"] = True

        report.append({
            "index": idx,
            "short": theme.get("short"),
            "version": theme.get("version"),
            "source": src.name,
            "output": final_name,
            "sha256": digest,
        })
        print(f"[OK] {idx:02d}/{len(themes)} {theme.get('short')} -> {final_name}", flush=True)

    (out / "theme-v1.rebranded.json").write_text(
        json.dumps(catalog, ensure_ascii=False, indent=2) + "\n",
        encoding="utf-8",
    )
    (out / "rebrand-report.json").write_text(
        json.dumps(report, ensure_ascii=False, indent=2) + "\n",
        encoding="utf-8",
    )

    print(f"Rebranded {len(report)}/{len(themes)} theme packages.")

if __name__ == "__main__":
    main()
