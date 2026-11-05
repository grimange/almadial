#!/usr/bin/env python3
"""M0-02 package/source evidence checks. Never installs packages or runs sources."""
from __future__ import annotations

import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile
from datetime import datetime, timezone

ROOT = Path(__file__).resolve().parents[1]
BASELINE = ROOT / "docs/compatibility/m0-02-candidate-stack.json"
VM = "almadial-alma10"
FORMAT = "%{name}|%{epoch}|%{version}|%{release}|%{arch}|%{repoid}"
FIELDS = ("name", "epoch", "version", "release", "arch", "repository")

# M0-02 is a historical qualification snapshot. M0-04 authenticated this exact
# unchanged tree through the official HTTPS nightly archive. This exception only
# checks the recorded identity; it does not itself authenticate a download.
M004_PROVENANCE = {
    "vicidial.provenance_state": "official_https_snapshot_byte_matched_to_pinned_export",
    "vicidial.revision": "4037",
    "vicidial.schema_version": "1748",
    "vicidial.exported_tree_sha256": "04a41c36c381981853cd0fc49a82fea9dda611cf39b336e9e8d92d459551eaed",
    "vicidial.authenticated_archive": "https://www.vicidial.org/svn_trunk_nightly/vicidial-trunk-2026-11-05.zip",
    "vicidial.archive_sha256": "ebdd6fab3c9bd56c5ed84f79a3118153c1c9ed38508f24a83a1b71ad803a4058",
    "vicidial.provenance_evidence": "docs/compatibility/m0-04-vicidial-runtime.md",
}


def parse_packages(text: str) -> list[dict[str, str]]:
    rows = []
    for line in text.splitlines():
        if not line.strip():
            continue
        values = line.strip().split("|")
        if len(values) != 6 or not all(values) or not values[1].isdigit():
            raise ValueError("malformed RPM query row")
        if values[4] not in {"x86_64", "noarch"}:
            raise ValueError("unexpected package architecture")
        rows.append(dict(zip(FIELDS, values)))
    return sorted(rows, key=lambda row: tuple(row[k] for k in FIELDS))


def tree_digest(directory: Path) -> str:
    """SHA256 of sorted 'file-SHA256  relative-posix-path\n' records, no metadata."""
    records = []
    for path in sorted(directory.rglob("*")):
        if path.is_symlink():
            raise ValueError("source tree contains a symlink")
        if path.is_file():
            records.append(hashlib.sha256(path.read_bytes()).hexdigest()
                           + "  " + path.relative_to(directory).as_posix() + "\n")
    if not records:
        raise ValueError("empty source tree")
    return hashlib.sha256("".join(records).encode()).hexdigest()


def manifest_leaves(text: str) -> dict[str, str]:
    """Read this manifest's intentionally small mapping/scalar subset, not general YAML."""
    stack: list[tuple[int, str]] = []
    leaves = {}
    seen = set()
    for line in text.splitlines():
        if not line.strip() or line.lstrip().startswith("#"):
            continue
        match = re.fullmatch(r"( *)([a-z0-9_]+):(?: (.*))?", line)
        if not match or len(match[1]) % 2:
            raise ValueError("unsupported manifest syntax")
        level, key, value = len(match[1]), match[2], match[3]
        while stack and stack[-1][0] >= level:
            stack.pop()
        if level != (stack[-1][0] + 2 if stack else 0):
            raise ValueError("invalid manifest indentation")
        name = ".".join([item[1] for item in stack] + [key])
        if name in seen:
            raise ValueError("duplicate manifest key")
        seen.add(name)
        if value is None:
            stack.append((level, key))
        else:
            leaves[name] = value.strip('"\'')
    return leaves


def check_local(baseline: dict, manifest: str) -> list[str]:
    issues = []
    if baseline["status"] != "candidate" or baseline["runtime_compatibility"] != "NOT_RUN":
        issues.append("baseline must remain candidate with runtime NOT_RUN")
    leaves = manifest_leaves(manifest)
    for key, expected in baseline["manifest_pins"].items():
        if leaves.get(key) != str(expected):
            if key == "vicidial.provenance_state" and all(
                    leaves.get(k) == v for k, v in M004_PROVENANCE.items()):
                continue
            issues.append(f"manifest mismatch: {key}")
    requested = [row["requested"] for row in baseline["packages"]]
    if not requested or len(requested) != len(set(requested)):
        issues.append("empty or duplicate dependency inventory")
    for source in baseline["sources"]:
        if not re.fullmatch(r"[0-9a-f]{64}", source["sha256"]):
            issues.append(f"missing source digest: {source['component']}")
        if not source["url"].startswith(("https://", "svn://")) or not source["license"]:
            issues.append(f"missing source attribution: {source['component']}")
    # The JSON pins are not independent evidence of its source/package rows.
    # Compare those rows directly with the actual manifest as well.
    source_contracts = {
        "Asterisk": ("asterisk", "sha256"),
        "PJPROJECT": ("pjproject", "sha256"),
        "Net::Telnet": ("perl_net_telnet", "sha256"),
        "VICIdial": ("vicidial", "exported_tree_sha256"),
        "VICIphone": ("webrtc", "sha256"),
    }
    for component, (prefix, digest_key) in source_contracts.items():
        matches = [s for s in baseline["sources"] if s["component"] == component]
        if len(matches) != 1:
            issues.append(f"source identity missing or duplicated: {component}")
            continue
        source = matches[0]
        if source["sha256"] != leaves.get(f"{prefix}.{digest_key}"):
            issues.append(f"source/manifest digest mismatch: {component}")
        if component == "VICIdial":
            version = re.fullmatch(r"SVN r(\d+); [^;]+; schema (\d+)", source["version"])
            if not version or version.groups() != (leaves.get("vicidial.revision"), leaves.get("vicidial.schema_version")):
                issues.append("source/manifest version mismatch: VICIdial")
            expected_url = leaves.get("vicidial.source", "") + "@" + leaves.get("vicidial.revision", "")
        elif component == "VICIphone":
            if source["version"].split("; ")[-1] != leaves.get("webrtc.revision"):
                issues.append("source/manifest version mismatch: VICIphone")
            expected_url = "https://codeload.github.com/vicimikec/ViciPhone/tar.gz/" + leaves.get("webrtc.revision", "")
        else:
            if source["version"].split("; ")[0] != leaves.get(f"{prefix}.version"):
                issues.append(f"source/manifest version mismatch: {component}")
            expected_url = leaves.get(f"{prefix}.source")
        if source["url"] != expected_url:
            issues.append(f"source/manifest URL mismatch: {component}")
    for prefix in ("mariadb", "php", "perl", "apache", "srtp", "jansson"):
        matches = [p for row in baseline["packages"] for p in row["selected"]
                   if f"{p['name']}-{p['epoch']}:{p['version']}-{p['release']}.{p['arch']}" == leaves.get(f"{prefix}.nevra")
                   and p["repository"] == leaves.get(f"{prefix}.repository")]
        if not matches:
            issues.append(f"package/manifest identity mismatch: {prefix}")
    for row in baseline["packages"]:
        if row["availability"] == "Available" and not row["selected"]:
            issues.append(f"availability without an exact package: {row['requested']}")
        for selected in row["selected"]:
            parse_packages("|".join(selected[k] for k in FIELDS))
    return issues


def check_matrix(baseline: dict, text: str) -> list[str]:
    issues = []
    for row in baseline["packages"]:
        prefix = "| " + row["requested"] + " |"
        matches = [line for line in text.splitlines() if line.startswith(prefix)]
        if len(matches) != 1:
            issues.append(f"matrix mapping missing or duplicated: {row['requested']}")
            continue
        line = matches[0]
        for package in row["selected"]:
            version = f"{package['epoch']}:{package['version']}-{package['release']}.{package['arch']}"
            if version not in line or package["repository"] not in line:
                issues.append(f"matrix RPM mismatch: {row['requested']}")
        if f"| {row['availability']} |" not in line or "| Not Tested |" not in line:
            issues.append(f"matrix state mismatch: {row['requested']}")
    return issues


def read_record(directory: Path, name: str) -> str:
    record = json.loads((directory / (name + ".json")).read_text())
    if record.get("exit_code") != 0:
        raise ValueError(f"{name}: command failed or unavailable; not package absence")
    return record["stdout"]


def check_evidence(baseline: dict, directory: Path) -> list[str]:
    issues = []
    os_release = dict(line.split("=", 1) for line in read_record(directory, "os").splitlines()
                      if "=" in line and not line.startswith("#"))
    for key, expected in (("ID", "almalinux"), ("VERSION_ID", "10.2")):
        if os_release.get(key, "").strip('"') != expected:
            issues.append(f"OS mismatch: {key}")
    if read_record(directory, "architecture").strip() != "x86_64":
        issues.append("architecture mismatch")
    repositories = {line.split()[0] for line in read_record(directory, "repositories").splitlines()
                    if line.strip()}
    for repo in baseline["enabled_repositories"]:
        if repo not in repositories:
            issues.append(f"repository unavailable: {repo}")
    rows = parse_packages(read_record(directory, "packages"))
    if not rows:
        issues.append("empty package metadata result")
    for item in baseline["packages"]:
        for selected in item["selected"]:
            if selected["repository"] != "epel-10.2-isolated" and selected not in rows:
                issues.append(f"pinned RPM unavailable: {item['requested']}")
        was_missing = item["requested"] in baseline["missing_in_enabled_repositories"]
        if was_missing and any(row["name"] == item["requested"] for row in rows):
            issues.append(f"previously missing package now available: {item['requested']}")
    if sorted(read_record(directory, "installed-before").splitlines()) != sorted(
            read_record(directory, "installed-after").splitlines()):
        issues.append("installed RPM set changed during qualification")
    return issues


def check_sources(baseline: dict, directory: Path) -> list[str]:
    issues = []
    for source in baseline["sources"]:
        path = directory / source["artifact"]
        if not path.exists():
            issues.append(f"source unavailable: {source['component']}")
            continue
        digest = tree_digest(path) if source.get("kind") == "tree" else hashlib.sha256(path.read_bytes()).hexdigest()
        if digest != source["sha256"]:
            issues.append(f"source digest mismatch: {source['component']}")
    return issues


def run_record(argv: list[str], timeout: int = 120) -> dict:
    record = {"argv": argv, "started_at": datetime.now(timezone.utc).isoformat()}
    # DNF output is restricted to explicit names. Spool diagnostics rather than holding
    # arbitrary command output in memory; reject excessive output before reading it.
    with tempfile.TemporaryFile() as stdout, tempfile.TemporaryFile() as stderr:
        try:
            result = subprocess.run(argv, stdout=stdout, stderr=stderr, timeout=timeout,
                                    check=False, env={**os.environ, "LC_ALL": "C"})
            record["exit_code"] = result.returncode
        except FileNotFoundError:
            record.update(exit_code=None, error="UNAVAILABLE")
        except subprocess.TimeoutExpired:
            record.update(exit_code=None, error="TIMEOUT")
        for label, stream in (("stdout", stdout), ("stderr", stderr)):
            size = stream.tell()
            stream.seek(0)
            record[label] = stream.read(2_000_000).decode(errors="replace")
            if size > 2_000_000:
                record.update(exit_code=None, error="OUTPUT_LIMIT")
    record["finished_at"] = datetime.now(timezone.utc).isoformat()
    return record


def collect(baseline: dict, output: Path, refresh: bool) -> None:
    output.mkdir(mode=0o700, parents=False, exist_ok=False)
    guest = ["multipass", "exec", VM, "--"]
    rpm = ["rpm", "-qa", "--qf", "%{NAME}|%{EPOCHNUM}|%{VERSION}|%{RELEASE}|%{ARCH}\\n"]
    names = sorted({row["requested"] for row in baseline["packages"]}
                   | {p["name"] for row in baseline["packages"] for p in row["selected"]
                      if p["repository"] != "epel-10.2-isolated"})
    if any(not re.fullmatch(r"[A-Za-z0-9_+.-]+", name) for name in names):
        raise ValueError("invalid package name")
    commands = [("installed-before", guest + rpm),
                ("os", guest + ["cat", "/etc/os-release"]),
                ("architecture", guest + ["uname", "-m"]),
                ("repositories", guest + ["sudo", "-n", "dnf", "-q", "--cacheonly", "repolist", "--enabled"])]
    if refresh:
        commands.append(("refresh", guest + ["sudo", "-n", "dnf", "makecache", "--refresh"]))
    commands.extend([
        ("packages", guest + ["sudo", "-n", "dnf", "-q", "--cacheonly", "repoquery", "--available",
                              "--showduplicates", "--qf", FORMAT] + names),
        ("installed-after", guest + rpm)])
    for name, argv in commands:
        record = run_record(argv)
        with (output / (name + ".json")).open("x", encoding="utf-8") as stream:
            json.dump(record, stream, indent=2)
            stream.write("\n")
        if record["exit_code"] != 0:
            raise ValueError(f"{name}: command failed; evidence preserved")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    mode = parser.add_mutually_exclusive_group()
    mode.add_argument("--live", action="store_true", help=f"query only {VM}")
    mode.add_argument("--replay", type=Path, help="read existing command records, execute nothing")
    parser.add_argument("--refresh", action="store_true", help="allow metadata refresh with --live")
    parser.add_argument("--output-dir", type=Path, help="new protected directory for --live")
    parser.add_argument("--source-dir", type=Path, help="verify previously acquired sources, never execute")
    args = parser.parse_args()
    if args.live != bool(args.output_dir) or (args.refresh and not args.live):
        parser.error("--live requires --output-dir; --refresh requires --live")
    os.umask(0o077)
    try:
        baseline = json.loads(BASELINE.read_text())
        issues = check_local(baseline, (ROOT / "manifests/stack.lock.yaml").read_text())
        issues.extend(check_matrix(baseline, (ROOT / "docs/compatibility/m0-02-dependency-matrix.md").read_text()))
        evidence = args.replay
        if args.live:
            collect(baseline, args.output_dir, args.refresh)
            evidence = args.output_dir
        if evidence:
            issues.extend(check_evidence(baseline, evidence))
        if args.source_dir:
            issues.extend(check_sources(baseline, args.source_dir))
        result = {"schema_version": 1, "mode": "LIVE" if args.live else "REPLAY" if args.replay else "LOCAL",
                  "checked_at": datetime.now(timezone.utc).isoformat(),
                  "candidate_consistency": "FAIL" if issues else "PASS", "issues": issues,
                  "package_evidence": "CHECKED" if evidence else "NOT_RUN",
                  "source_hashes": "CHECKED" if args.source_dir else "NOT_RUN",
                  "runtime_compatibility": "NOT_RUN", "signature_and_origin_trust": "SEE_PROVENANCE_DECISIONS"}
        print(json.dumps(result, indent=2))
        return 1 if issues else 0
    except (OSError, ValueError, KeyError, TypeError) as error:
        print(json.dumps({"candidate_consistency": "FAIL", "error": str(error), "runtime_compatibility": "NOT_RUN"}))
        return 1


if __name__ == "__main__":
    sys.exit(main())
