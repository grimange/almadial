#!/usr/bin/env python3
"""Derive reviewed r4037 lab inputs; never execute SQL or change upstream files."""
import argparse
import hashlib
import json
import os
from pathlib import Path


PINS = {
    "extras/MySQL_AST_CREATE_tables.sql": "8a210454a6f42fd3ef0054033a5c4396a0cd3615442ebd53491dfd2efa38de40",
    "bin/ADMIN_keepalive_ALL.pl": "4406700454ec6b9803393cbf9f3672bca6b1123d01c3b5b58ef00cb1ec2c0b67",
    "bin/AST_manager_send.pl": "83be4e5ca25591adc6991273b84114b53164ef49b7bf04bee9e9b7413b7cb384",
    "bin/AST_VDauto_dial.pl": "cd2115f06d8c7730af73d67d468612c4ac127f815b31689a3112b374a14d9d87",
    "bin/AST_VDremote_agents.pl": "0187ca67f99c0b435280fe76a2cfd2c4816ad7f82411f2112bc988c37ee4a596",
}
LEGACY_PS = b'/bin/ps -o "%p %a" --no-headers -A'
WIDE_PS = b'/bin/ps -o pid,args -ww --no-headers -A'
PRIVILEGES = (b"GRANT RELOAD ON *.* TO cron@'%';",
              b"GRANT RELOAD ON *.* TO cron@localhost;", b"flush privileges;")


def sha256(data):
    return hashlib.sha256(data).hexdigest()


def transform(name, data):
    """Refuse different revisions or already transformed inputs."""
    if name not in PINS or sha256(data) != PINS[name]:
        raise ValueError(f"unreviewed source bytes: {name}")
    if name.endswith(".sql"):
        for statement in PRIVILEGES:
            if data.count(statement) != 1:
                raise ValueError("unexpected privilege statement count")
            data = data.replace(statement, b"-- AlmaDial M0-04: privilege provisioning owned by lab bootstrap.")
    else:
        expected = 3 if name.endswith("ADMIN_keepalive_ALL.pl") else 1
        if data.count(LEGACY_PS) != expected:
            raise ValueError("unexpected process query count")
        data = data.replace(LEGACY_PS, WIDE_PS)
    return data


def prepare(source, output):
    source, output = Path(source), Path(output)
    # Validate every input before producing anything. Do not follow source links.
    derived = {}
    for name in PINS:
        path = source / name
        if path.is_symlink() or any(p.is_symlink() for p in path.parents):
            raise ValueError(f"source symlink: {name}")
        derived[name] = transform(name, path.read_bytes())
    output.mkdir(mode=0o700)  # Exclusive; never reuse an evidence destination.
    ledger = {"revision": 4037, "schema": 1748, "execution": "NOT_RUN",
              "purpose": "lab inputs only; does not deploy or import", "files": []}
    for name, data in derived.items():
        target = output / name
        target.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
        fd = os.open(target, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
        with os.fdopen(fd, "wb") as stream:
            stream.write(data)
        ledger["files"].append({"path": name, "source_sha256": PINS[name],
                                "derived_sha256": sha256(data)})
    with (output / "derivation.json").open("x", encoding="utf-8") as stream:
        os.chmod(stream.name, 0o600)
        json.dump(ledger, stream, indent=2)
        stream.write("\n")
    return ledger


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source-dir", type=Path, required=True)
    parser.add_argument("--output-dir", type=Path, required=True)
    args = parser.parse_args()
    try:
        result = prepare(args.source_dir, args.output_dir)
    except (OSError, ValueError) as error:
        parser.exit(1, f"refused: {error}\n")
    print(json.dumps(result, indent=2))


if __name__ == "__main__":
    main()
