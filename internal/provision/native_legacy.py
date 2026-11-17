"""Fixed DNF 4/RPM adapter, embedded in the Go stage; never a shell or plugin API.

inspect reads RPM/config/history only. resolve explicitly permits native metadata
cache writes. apply re-solves and rejects every change outside the reviewed full
transaction before downloading/signature checking and calling native DNF.
"""
import base64
import configparser
from contextlib import closing
import hashlib
import json
import logging
import os
import pathlib
import re
import sqlite3
import sys
import traceback

import dnf
import hawkey
import rpm

# No DNF exception, plugin or proxy diagnostics bypass the Go logger's boundary.
logging.disable(logging.CRITICAL)


def digest(data):
    return hashlib.sha256(data).hexdigest()


def identity(pkg):
    return dict(name=pkg.name, evr=str(pkg.epoch or 0) + ":" + pkg.version + "-" + pkg.release,
                arch=pkg.arch, repository=pkg.reponame)


def file_hash(path):
    return digest(pathlib.Path(path).read_bytes())


def base_config():
    base = dnf.Base()
    base.conf.read()
    directories = ["/etc/yum.repos.d", "/etc/yum/repos.d", "/etc/distro.repos.d"]
    if base.conf.installroot != "/" or list(base.conf.reposdir) != directories:
        raise ValueError("configuration")
    # Read definitions with strict duplicate detection before DNF's merge behavior.
    names = set()
    paths = [p for directory in directories for p in pathlib.Path(directory).glob("*.repo")]
    for path in sorted(paths):
        parser = configparser.ConfigParser(interpolation=None, strict=True)
        parser.read_string(path.read_text())
        for name in parser.sections():
            if name in names:
                raise ValueError("duplicate repository")
            names.add(name)
    base.conf.plugins = False
    base.read_all_repos()
    return base


def history():
    path = pathlib.Path("/var/lib/dnf/history.sqlite")
    if not path.exists():
        return digest(b"[]"), True
    # SQLite's supported read-only URI reads committed WAL data. Refuse to create
    # a missing shared-memory index; inspection must never initialize DNF state.
    wal = pathlib.Path(str(path) + "-wal")
    shm = pathlib.Path(str(path) + "-shm")
    if wal.exists() and wal.stat().st_size and not shm.exists():
        raise ValueError("history requires reconciliation")
    def state():
        return digest(path.read_bytes() + (wal.read_bytes() if wal.exists() else b""))
    before = state()
    with closing(sqlite3.connect("file:" + str(path) + "?mode=ro", uri=True)) as db:
        db.execute("PRAGMA query_only=ON")
        rows = db.execute("SELECT id,dt_begin,dt_end,state FROM trans ORDER BY id").fetchall()
    if state() != before:
        raise ValueError("history changed")
    # DNF may initialize/checkpoint this database during explicit resolution.
    # Bind plans to transaction content, not SQLite's incidental file layout.
    return digest(json.dumps(rows, separators=(",", ":")).encode()), all(row[2] is not None and row[3] == 1 for row in rows)


def inspect():
    release = {}
    for line in pathlib.Path("/etc/os-release").read_text().splitlines():
        key, sep, value = line.partition("=")
        if sep:
            release[key] = value.strip('"')
    installed, keys = [], []
    ts = rpm.TransactionSet()
    for hdr in ts.dbMatch():
        signature = hdr.sprintf("%{RSAHEADER:pgpsig}")
        match = re.search(r"Key ID ([0-9a-fA-F]{16})$", signature)
        installed.append(dict(name=hdr["name"], evr=str(hdr["epoch"] or 0) + ":" + hdr["version"] + "-" + hdr["release"],
                              arch=str(hdr["arch"] or "(none)"), signer=match.group(1).lower() if match else ""))
        if hdr["name"] == "gpg-pubkey":
            keys.append(digest(hdr["description"].encode()))
    with base_config() as base:
        repos = [dict(id=r.id, enabled=bool(r.enabled), mirrorlist=r.mirrorlist or "", metalink=r.metalink or "",
                      baseurls=list(r.baseurl), keys=list(r.gpgkey), gpgcheck=bool(r.gpgcheck),
                      tls_verify=bool(r.sslverify), repo_gpgcheck=bool(r.repo_gpgcheck)) for r in base.repos.values()]
    hist, complete = history()
    return dict(host=digest(pathlib.Path("/etc/machine-id").read_bytes().strip()), os=release.get("ID", ""),
                release=release.get("VERSION_ID", ""), arch=os.uname().machine, repositories=repos,
                installed=installed, key_hashes=keys,
                alma_key_file_sha256=file_hash("/etc/pki/rpm-gpg/RPM-GPG-KEY-AlmaLinux-10"),
                history_sha256=hist, history_complete=complete)


def canonical_snapshot(s):
    s = dict(s)
    s["repositories"] = sorted(s["repositories"], key=lambda r: r["id"])
    for r in s["repositories"]:
        r["baseurls"], r["keys"] = sorted(r["baseurls"]), sorted(r["keys"])
    s["installed"] = sorted(s["installed"], key=lambda p: p["name"] + "-" + p["evr"] + "." + p["arch"])
    s["key_hashes"] = sorted(s["key_hashes"])
    return s


def solve(request):
    with base_config() as base:
        base.conf.assumeyes = True
        base.conf.gpgcheck = True
        base.conf.localpkg_gpgcheck = True
        base.conf.sslverify = True
        base.conf.install_weak_deps = False
        base.conf.skip_if_unavailable = False
        base.conf.best = True
        base.conf.exit_on_lock = True
        base.conf.substitutions["releasever"] = "10"
        # No persistent .repo changes or key imports. The only EPEL packages
        # visible to the solver are the six qualified, pinned M0 Perl RPMs.
        for repo in base.repos.values():
            if repo.id not in ("appstream", "baseos", "crb", "extras"):
                repo.disable()
            repo.skip_if_unavailable = False
        epel = dnf.repo.Repo("epel-almadial", base.conf)
        epel.baseurl = ["https://dl.fedoraproject.org/pub/epel/10.2/Everything/x86_64/"]
        epel.gpgcheck, epel.sslverify = True, True
        epel.skip_if_unavailable = False
        epel.includepkgs = [p["name"] for p in request["packages"] if p["repository"] == "epel-almadial"]
        base.repos.add(epel)
        base.fill_sack()
        for p in request["packages"]:
            nevra = p["name"] + "-" + p["evr"] + "." + p["arch"]
            base.install(nevra, reponame=p["repository"], strict=True)
        base.resolve(allow_erasing=False)
        installs = []
        for p in base.transaction.install_set:
            kind, value = p.chksum
            if kind != hawkey.CHKSUM_SHA256:
                raise ValueError("checksum")
            installs.append(dict(identity(p), sha256=bytes(value).hex()))
        txn = dict(before_sha256=request["before_sha256"],
                   install=sorted(installs, key=lambda p: p["name"] + "-" + p["evr"] + "." + p["arch"] + p["repository"]),
                   remove=sorted([identity(p) for p in base.transaction.remove_set], key=lambda p: p["name"]))
        if request["mode"] == "apply":
            # RPM, rather than a Go/string version comparator, establishes
            # direction. Go separately restricts these to seven exact updates.
            for old in txn["remove"]:
                pairs = [p for p in installs if p["name"] == old["name"] and p["arch"] == old["arch"]]
                if len(pairs) != 1 or rpm.labelCompare(evr_tuple(pairs[0]["evr"]), evr_tuple(old["evr"])) <= 0:
                    raise ValueError("replacement")
        existing = {p["name"] for p in request["before"]["installed"]}
        replaced = {p["name"] for p in txn["remove"]}
        for p in installs:
            if (p["name"] in existing and p["name"] not in replaced and request["mode"] == "apply") or p["arch"] not in ("x86_64", "noarch") or ".el10" not in p["evr"]:
                raise ValueError("scope")
            if p["repository"] not in ("appstream", "baseos", "crb", "extras", "epel-almadial"):
                raise ValueError("origin")
            if p["repository"] == "epel-almadial" and identity_fields(p) not in request["packages"]:
                raise ValueError("epel scope")
        if request["mode"] == "apply":
            if txn != request["transaction"]:
                raise ValueError("stale transaction")
            base.download_packages(base.transaction.install_set)
            for p in base.transaction.install_set:
                if file_hash(p.localPkg()) != next(c["sha256"] for c in installs if c["name"] == p.name):
                    raise ValueError("payload checksum")
                # No implicit key import: only already reviewed RPM keyring trust.
                if base.package_signature_check(p)[0] != 0:
                    raise ValueError("signature")
                with open(p.localPkg(), "rb") as handle:
                    hdr = rpm.TransactionSet().hdrFromFdno(handle.fileno())
                expected = "33d98517e37ed158" if p.reponame == "epel-almadial" else "dee5c11cc2a1e572"
                if not hdr.sprintf("%{RSAHEADER:pgpsig}").lower().endswith("key id " + expected):
                    raise ValueError("signer")
            if canonical_snapshot(inspect()) != canonical_snapshot(request["before"]):
                raise ValueError("stale host")
            if installs:
                base.do_transaction()  # native checks, RPM lock, test, scriptlets and DNF history
        return txn


def evr_tuple(evr):
    epoch, rest = evr.split(":", 1)
    version, release = rest.rsplit("-", 1)
    return epoch, version, release


def identity_fields(p):
    return {k: p[k] for k in ("name", "evr", "arch", "repository")}


def import_key(request):
    key = request["public_key"]
    if digest(key.encode()) != "de390fc168eae5ab2852e9e93d34a0b9ddf05cf9ce90ee28d97de26a4b1f6b93":
        raise ValueError("untrusted public key")
    # Decode only the pinned ASCII armor, not arbitrary user key material.
    body = key.split("\n\n", 1)[1]
    encoded = "".join(line for line in body.splitlines() if line and not line.startswith(("=", "-")))
    if rpm.TransactionSet().pgpImportPubkey(base64.b64decode(encoded, validate=True)) != 0:
        raise ValueError("key import")
    return {"key_sha256": digest(key.encode())}


def main():
    mode = sys.argv[1]
    if mode == "inspect":
        result = inspect()
    else:
        request = json.loads(sys.argv[2])
        if mode not in ("resolve", "apply", "trust") or request["mode"] != mode:
            raise ValueError("mode")
        if canonical_snapshot(inspect()) != canonical_snapshot(request["before"]):
            raise ValueError("stale host")
        result = import_key(request) if mode == "trust" else solve(request)
    print(json.dumps(result, separators=(",", ":")))


if __name__ == "__main__":
    try:
        main()
    except Exception as error:
        # DNF exceptions can contain private proxy URLs and configuration paths.
        # Go retains a structured nonzero exit; never export raw exception text.
        # Class and adapter line numbers aid API diagnosis without exporting
        # exception values (which can contain credentials/private source URLs).
        lines = [str(frame.lineno) for frame in traceback.extract_tb(error.__traceback__) if frame.filename == "<string>"]
        print("RPM operation rejected (" + type(error).__name__ + "; adapter lines " + ",".join(lines) + "); inspect trust, availability and transaction state", file=sys.stderr)
        sys.exit(1)
