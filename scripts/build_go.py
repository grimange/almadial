#!/usr/bin/env python3
"""Bounded reproducible CLI build. No compiler installation or network access."""
import argparse
import os
from pathlib import Path
import re
import subprocess

ROOT = Path(__file__).resolve().parent.parent


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", default="dist/almadial")
    args = parser.parse_args()
    env = dict(
        os.environ,
        GOTOOLCHAIN="local",
        GOENV="off",
        GOFLAGS="",
        GOOS="linux",
        GOARCH="amd64",
        GOAMD64="v1",
        CGO_ENABLED="0",
        GOPROXY="off",
        GOSUMDB="off",
        GOWORK="off",
        GOEXPERIMENT="",
    )
    go = os.environ.get("GO", "go")
    pin = (ROOT / ".go-version").read_text().strip()
    actual = subprocess.check_output([go, "env", "GOVERSION"], env=env, text=True, timeout=30).strip()
    if actual != "go" + pin:
        parser.error("compiler must be go" + pin + "; found " + actual)
    version = os.environ.get("ALMADIAL_VERSION", "dev")
    commit = os.environ.get("ALMADIAL_COMMIT")
    if commit is None:
        commit = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True, timeout=30).strip()
        if subprocess.check_output(["git", "status", "--porcelain"], cwd=ROOT, timeout=30):
            commit += "-dirty"
    for value in (version, commit):
        if not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._+-]{0,127}", value):
            parser.error("metadata must contain 1-128 ASCII letters/digits/dot/underscore/plus/hyphen")
    output = Path(args.output).resolve()
    output.parent.mkdir(parents=True, exist_ok=True)
    package = "github.com/grimange/almadial/internal/buildinfo"
    flags = f"-X {package}.Version={version} -X {package}.Commit={commit}"
    subprocess.run(
        [go, "build", "-mod=readonly", "-trimpath", "-buildvcs=false",
         "-ldflags=" + flags, "-o", str(output), "./cmd/almadial"],
        cwd=ROOT, env=env, check=True, timeout=300,
    )


if __name__ == "__main__":
    main()
