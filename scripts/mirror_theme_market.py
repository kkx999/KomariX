#!/usr/bin/env python3
import argparse
import hashlib
import json
import mimetypes
import os
import re
import shutil
import sys
import urllib.parse
import urllib.request
from pathlib import Path

UA = "KomariX-Theme-Mirror/1.0 (+https://github.com/kkx999/KomariX)"

def slug(value: str) -> str:
    value = re.sub(r"[^A-Za-z0-9._-]+", "-", str(value or "").strip())
    value = value.strip("-.")
    return value[:80] or "theme"

def download(url: str, dest: Path) -> tuple[str, str]:
    req = urllib.request.Request(url, headers={
        "User-Agent": UA,
        "Accept": "*/*",
    })
    h = hashlib.sha256()
    content_type = ""
    with urllib.request.urlopen(req, timeout=120) as resp, dest.open("wb") as out:
        content_type = (resp.headers.get("Content-Type") or "").split(";", 1)[0].strip().lower()
        while True:
            chunk = resp.read(1024 * 1024)
            if not chunk:
                break
            out.write(chunk)
            h.update(chunk)
    return h.hexdigest(), content_type

def preview_ext(url: str, content_type: str, temp: Path) -> str:
    suffix = Path(urllib.parse.urlparse(url).path).suffix.lower()
    if suffix in {".png", ".jpg", ".jpeg", ".webp", ".gif", ".svg", ".avif"}:
        return suffix
    mapping = {
        "image/png": ".png",
        "image/jpeg": ".jpg",
        "image/webp": ".webp",
        "image/gif": ".gif",
        "image/svg+xml": ".svg",
        "image/avif": ".avif",
    }
    if content_type in mapping:
        return mapping[content_type]
    guessed = mimetypes.guess_extension(content_type) if content_type else None
    return guessed or ".img"

def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--catalog", default="market/theme-v1.json")
    ap.add_argument("--output-dir", required=True)
    ap.add_argument("--release-tag", required=True)
    ap.add_argument("--owner", default="kkx999")
    ap.add_argument("--repo", default="KomariX")
    args = ap.parse_args()

    catalog_path = Path(args.catalog)
    raw = json.loads(catalog_path.read_text(encoding="utf-8"))
    themes = raw if isinstance(raw, list) else raw.get("themes", [])
    out = Path(args.output_dir)
    out.mkdir(parents=True, exist_ok=True)

    rewritten = json.loads(json.dumps(raw))
    target_themes = rewritten if isinstance(rewritten, list) else rewritten["themes"]

    base = f"https://github.com/{args.owner}/{args.repo}/releases/download/{args.release_tag}"
    report = []
    failures = []

    for idx, (src, dst) in enumerate(zip(themes, target_themes), start=1):
        short = slug(src.get("short", f"theme-{idx}"))
        version = slug(src.get("version", "unknown"))
        expected = str(src.get("sha256") or "").lower().removeprefix("sha256:")
        download_url = str(src.get("download") or "")
        preview_url = str(src.get("preview") or "")

        if not download_url or not expected:
            failures.append(f"{short}: missing download or sha256")
            continue

        zip_name = f"theme-{idx:02d}-{short}-{version}-{expected[:12]}.zip"
        zip_path = out / zip_name
        try:
            actual, zip_type = download(download_url, zip_path)
            if actual.lower() != expected:
                raise RuntimeError(f"SHA256 mismatch expected={expected} actual={actual}")
        except Exception as exc:
            failures.append(f"{short}: package download failed: {exc}")
            if zip_path.exists():
                zip_path.unlink()
            continue

        dst["upstream_download"] = download_url
        dst["download"] = f"{base}/{zip_name}"

        preview_name = ""
        preview_sha = ""
        if preview_url:
            preview_tmp = out / f".preview-{idx:02d}.tmp"
            try:
                preview_sha, preview_type = download(preview_url, preview_tmp)
                ext = preview_ext(preview_url, preview_type, preview_tmp)
                preview_name = f"preview-{idx:02d}-{short}-{version}-{preview_sha[:12]}{ext}"
                preview_path = out / preview_name
                preview_tmp.replace(preview_path)
                dst["upstream_preview"] = preview_url
                dst["preview"] = f"{base}/{preview_name}"
                dst["preview_sha256"] = preview_sha
            except Exception as exc:
                failures.append(f"{short}: preview download failed: {exc}")
                if preview_tmp.exists():
                    preview_tmp.unlink()
                continue

        report.append({
            "short": src.get("short"),
            "version": src.get("version"),
            "package": zip_name,
            "package_sha256": expected,
            "preview": preview_name,
            "preview_sha256": preview_sha,
        })
        print(f"[OK] {idx:02d}/{len(themes)} {src.get('short')} {src.get('version')}", flush=True)

    (out / "theme-v1.mirrored.json").write_text(
        json.dumps(rewritten, ensure_ascii=False, indent=2) + "\n",
        encoding="utf-8",
    )
    (out / "mirror-report.json").write_text(
        json.dumps(report, ensure_ascii=False, indent=2) + "\n",
        encoding="utf-8",
    )

    if failures:
        print("\nMirror failures:", file=sys.stderr)
        for failure in failures:
            print(f" - {failure}", file=sys.stderr)
        return 1

    if len(report) != len(themes):
        print(f"Expected {len(themes)} mirrored themes, got {len(report)}", file=sys.stderr)
        return 1

    print(f"Mirrored {len(report)}/{len(themes)} themes successfully.")
    return 0

if __name__ == "__main__":
    raise SystemExit(main())
