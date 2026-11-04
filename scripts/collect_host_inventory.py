#!/usr/bin/env python3
"""M0 research inventory: explicit local collection or strictly offline fixture replay.

No shell, sudo, package changes, remote connections or raw output export. Python
3.12+ standard library only. This is not part of the future Go installer.
"""
from __future__ import annotations

import argparse
import configparser
import datetime as dt
import hashlib
import hmac
import ipaddress
import json
import os
from pathlib import Path
import re
import secrets
import selectors
import subprocess
import sys
import time

LIMIT = 1024 * 1024
PACKAGES = (
    "almalinux-release", "kernel", "glibc", "rpm", "dnf", "python3",
    "asterisk", "mariadb-server", "MariaDB-server", "httpd", "php",
    "php-fpm", "perl", "chrony", "firewalld", "selinux-policy",
)
UNITS = ("asterisk.service", "mariadb.service", "mysqld.service",
         "httpd.service", "php-fpm.service", "firewalld.service", "chronyd.service")
BINARY_PATHS = ("/usr/sbin/asterisk", "/usr/local/sbin/asterisk",
                "/usr/sbin/mariadbd", "/usr/sbin/mysqld", "/usr/sbin/httpd")
COMMANDS = {
    "kernel": ("uname", "-r"),
    "cpu": ("lscpu", "--json"),
    "disks": ("lsblk", "--json", "--bytes", "--output", "NAME,TYPE,SIZE,FSTYPE,MOUNTPOINTS"),
    "mounts": ("findmnt", "--json", "--bytes", "--output", "TARGET,SOURCE,FSTYPE,SIZE,AVAIL"),
    "interfaces": ("ip", "-json", "address", "show"),
    "routes_v4": ("ip", "-json", "-4", "route", "show", "table", "all"),
    "routes_v6": ("ip", "-json", "-6", "route", "show", "table", "all"),
    "hostname": ("hostname",),
    "packages": ("rpm", "-q", "--qf", "%{NAME}\t%{EPOCHNUM}:%{VERSION}-%{RELEASE}\t%{ARCH}\n", *PACKAGES),
    "selinux": ("getenforce",),
    "firewalld": ("firewall-cmd", "--state"),
    "firewall_zones": ("firewall-cmd", "--get-active-zones"),
    "firewall_rules": ("firewall-cmd", "--list-all-zones"),
    "time": ("timedatectl", "show", "--property=NTPSynchronized,Timezone,LocalRTC"),
    "chrony": ("chronyc", "-n", "tracking"),
    "services": ("systemctl", "show", "--no-pager", "--property=Id,LoadState,ActiveState,SubState,UnitFileState", *UNITS),
    "listeners": ("ss", "--no-header", "--listening", "--numeric", "--tcp", "--udp"),
    "virtualization": ("systemd-detect-virt",),
}
FILES = {"os": "/etc/os-release", "memory": "/proc/meminfo", "dns": "/etc/resolv.conf"}
PROBES = (*FILES, "repositories", "binaries", *COMMANDS)


def utc() -> str:
    return dt.datetime.now(dt.timezone.utc).isoformat(timespec="seconds").replace("+00:00", "Z")


class Redactor:
    """Opaque per-run identities. Key is deliberately never exported."""

    def __init__(self, key: bytes | None = None):
        self.key = key or secrets.token_bytes(32)

    def token(self, value: object) -> str:
        return "redacted-" + hmac.new(self.key, str(value).encode(), hashlib.sha256).hexdigest()[:20]


def enum(value: object, allowed: str) -> str:
    return str(value) if str(value) in allowed.split() else "unknown"


def number(value: object) -> int | None:
    if isinstance(value, bool):
        return None
    try:
        n = int(str(value))
        return n if n >= 0 else None
    except ValueError:
        return None


def version(value: object) -> str:
    s = str(value)
    if not re.fullmatch(r"[0-9][0-9A-Za-z._+:~^-]{0,127}", s):
        raise ValueError("invalid version")
    return s


def address(value: str, redactor: Redactor) -> dict:
    addr = ipaddress.ip_address(value.split("%", 1)[0])
    return {"identity": redactor.token(str(addr)), "family": addr.version,
            "loopback": addr.is_loopback, "link_local": addr.is_link_local,
            "private": addr.is_private, "unspecified": addr.is_unspecified}


def pairs(text: str) -> dict[str, str]:
    return dict(line.split("=", 1) for line in text.splitlines() if "=" in line)


def array(value: object) -> list:
    if not isinstance(value, list):
        raise ValueError("expected JSON array")
    return value


def require_fields(value: object, required: tuple[str, ...]) -> dict:
    if not isinstance(value, dict) or not set(required) <= value.keys():
        raise ValueError("missing required fields")
    return value


def required_enum(value: object, allowed: str) -> str:
    result = enum(value, allowed)
    if result == "unknown":
        raise ValueError("unrecognized required state")
    return result


def parse(probe: str, text: str, r: Redactor) -> dict | list:
    if probe == "os":
        p = {k: v.strip().strip('"\'') for k, v in pairs(text).items()}
        if p.get("ID") != "almalinux" or not re.fullmatch(r"10(?:\.\d+)*", p.get("VERSION_ID", "")):
            raise ValueError("requires AlmaLinux 10")
        return {"distribution": "AlmaLinux", "id": "almalinux", "version": p["VERSION_ID"]}
    if probe == "kernel":
        return {"release": version(text.strip())}
    if probe == "cpu":
        p = {x["field"].rstrip(":"): x["data"] for x in array(json.loads(text)["lscpu"])}
        flags = str(p.get("Flags", "")).split()
        if not flags or any(not re.fullmatch(r"[a-z0-9_]+", f) for f in flags):
            raise ValueError("invalid flags")
        if not number(p.get("CPU(s)")) or p.get("Architecture") not in {"x86_64", "aarch64", "ppc64le", "s390x"}:
            raise ValueError("missing CPU identity")
        return {"architecture": enum(p.get("Architecture"), "x86_64 aarch64 ppc64le s390x"),
                "cpu_count": number(p.get("CPU(s)")), "flags": sorted(set(flags)),
                "isa_qualification": "NOT_RUN"}
    if probe == "memory":
        p = dict(re.findall(r"^(MemTotal|MemAvailable):\s+(\d+) kB$", text, re.M))
        if set(p) != {"MemTotal", "MemAvailable"} or int(p["MemTotal"]) == 0:
            raise ValueError("missing memory")
        return {k + "_bytes": int(v) * 1024 for k, v in p.items()}
    if probe in {"disks", "mounts"}:
        def nodes(items: list) -> list:
            result = []
            for x in array(items):
                require_fields(x, ("name", "type", "size", "fstype", "mountpoints") if probe == "disks"
                       else ("source", "target", "fstype", "size", "avail"))
                identity = x["name" if probe == "disks" else "source"]
                if not isinstance(identity, str) or not identity:
                    raise ValueError("missing device identity")
                for key in (("size",) if probe == "disks" else ("size", "avail")):
                    if x[key] is not None and number(x[key]) is None:
                        raise ValueError("invalid storage metric")
                if probe == "disks":
                    required_enum(x["type"], "disk part lvm crypt rom loop raid1 raid0 raid5 raid6 raid10 mpath")
                    if not isinstance(x["mountpoints"], list) or any(v is not None and not isinstance(v, str) for v in x["mountpoints"]):
                        raise ValueError("invalid mountpoints")
                elif not isinstance(x["target"], str) or not x["target"]:
                    raise ValueError("missing mount target")
                entry = {"identity": r.token(x.get("name", x.get("source", "unknown"))),
                         "size_bytes": number(x.get("size")),
                         "filesystem": enum(x.get("fstype"), "xfs ext4 ext3 ext2 btrfs vfat tmpfs devtmpfs swap iso9660 overlay proc sysfs cgroup2 nfs nfs4")}
                if probe == "disks":
                    entry.update(type=enum(x.get("type"), "disk part lvm crypt rom loop raid1 raid0 raid5 raid6 raid10 mpath"),
                                 mountpoints=[r.token(v) for v in (x.get("mountpoints") or []) if v is not None])
                else:
                    entry.update(target=r.token(x.get("target")), available_bytes=number(x.get("avail")))
                if "children" in x:
                    entry["children"] = nodes(x["children"])
                result.append(entry)
            return result
        return nodes(json.loads(text)["blockdevices" if probe == "disks" else "filesystems"])
    if probe == "interfaces":
        interfaces = array(json.loads(text))
        for x in interfaces:
            require_fields(x, ("ifname", "mtu", "operstate", "addr_info"))
            if not isinstance(x["ifname"], str) or not x["ifname"] or not number(x["mtu"]):
                raise ValueError("missing interface identity")
            required_enum(x["operstate"], "UP DOWN UNKNOWN DORMANT LOWERLAYERDOWN NOTPRESENT TESTING")
            for a in array(x["addr_info"]):
                require_fields(a, ("local", "prefixlen", "scope"))
                bits = ipaddress.ip_address(a["local"].split("%", 1)[0]).max_prefixlen
                if number(a["prefixlen"]) is None or number(a["prefixlen"]) > bits:
                    raise ValueError("invalid address prefix")
                required_enum(a["scope"], "global link host site")
        return [{"identity": r.token(x["ifname"]), "mtu": number(x.get("mtu")),
                 "state": enum(x.get("operstate"), "UP DOWN UNKNOWN DORMANT LOWERLAYERDOWN NOTPRESENT TESTING"),
                 "addresses": [{**address(a["local"], r), "prefix_length": number(a.get("prefixlen")),
                                "scope": enum(a.get("scope"), "global link host site")} for a in x.get("addr_info", [])]}
                for x in interfaces]
    if probe.startswith("routes_"):
        routes = array(json.loads(text))
        family = 4 if probe == "routes_v4" else 6
        for x in routes:
            require_fields(x, ("dst",))
            if not isinstance(x["dst"], str) or not x["dst"]:
                raise ValueError("missing route destination")
            if x["dst"] != "default" and ipaddress.ip_network(x["dst"], strict=False).version != family:
                raise ValueError("invalid route family")
            route_type = required_enum(x.get("type", "unicast"), "unicast local broadcast multicast unreachable blackhole prohibit throw")
            if route_type in {"unicast", "local", "broadcast", "multicast"} or "dev" in x:
                if not isinstance(x.get("dev"), str) or not x["dev"]:
                    raise ValueError("missing route interface")
            if x.get("gateway") and address(x["gateway"], r)["family"] != family:
                raise ValueError("invalid gateway family")
            if "metric" in x and number(x["metric"]) is None:
                raise ValueError("invalid route metric")
        return [{"destination": "default" if x.get("dst") == "default" else r.token(x.get("dst")),
                 "gateway": address(x["gateway"], r) if x.get("gateway") else None,
                 "interface": r.token(x["dev"]) if x.get("dev") else None, "table": r.token(x.get("table", "main")),
                 "metric": number(x.get("metric")),
                 "type": enum(x.get("type", "unicast"), "unicast local broadcast multicast unreachable blackhole prohibit throw")}
                for x in routes]
    if probe == "hostname":
        if not text.strip():
            raise ValueError("empty hostname")
        return {"identity": r.token(text.strip())}
    if probe == "dns":
        servers, domains = [], []
        for line in text.splitlines():
            fields = re.split(r"\s+", re.split(r"[#;]", line, maxsplit=1)[0].strip())
            if fields[0] == "nameserver" and len(fields) == 2:
                servers.append(address(fields[1], r))
            if fields[0] in {"search", "domain"}:
                domains.extend(r.token(v) for v in fields[1:])
        return {"nameservers": servers, "search_domains": domains,
                "resolution_test": "NOT_RUN", "source": "resolv.conf"}
    if probe == "repositories":
        value = json.loads(text)
        conf = configparser.ConfigParser(interpolation=None)
        conf.read_string(value["dnf_config"])
        defaults = dict(conf["main"]) if conf.has_section("main") else {}
        result = []
        for content in value["configs"]:
            conf = configparser.ConfigParser(interpolation=None)
            conf.read_string(content)
            for name in conf.sections():
                s = conf[name]
                def flag(k: str, default: str | None = None) -> bool | None:
                    raw = s.get(k, defaults.get(k, default))
                    return True if raw in {"1", "true", "yes"} else False if raw in {"0", "false", "no"} else None
                result.append({"identity": r.token(name), "configured_enabled": flag("enabled", "1"),
                               "gpgcheck": flag("gpgcheck"), "repo_gpgcheck": flag("repo_gpgcheck", "0"),
                               "location_configured": any(s.get(k) for k in ("baseurl", "metalink", "mirrorlist")),
                               "signature_and_origin_verification": "NOT_RUN"})
        return {"repositories": result, "basis": "local_config_only", "metadata_or_remote_validation": "NOT_RUN"}
    if probe == "packages":
        installed, absent = [], []
        for line in text.splitlines():
            fields = line.split("\t")
            if len(fields) == 3 and fields[0] in PACKAGES:
                installed.append({"name": fields[0], "version": version(fields[1]),
                                  "architecture": enum(fields[2], "x86_64 x86_64_v2 noarch aarch64 ppc64le s390x")})
            elif line.startswith("package ") and line.endswith(" is not installed") and line[8:-17] in PACKAGES:
                absent.append(line[8:-17])
            elif line.strip():
                raise ValueError("unrecognized RPM query result")
        if {x["name"] for x in installed} | set(absent) != set(PACKAGES):
            raise ValueError("incomplete RPM query")
        return {"installed": installed, "not_installed": sorted(absent), "source_builds": "NOT_RUN"}
    if probe in {"selinux", "firewalld", "virtualization"}:
        allowed = {"selinux": "Enforcing Permissive Disabled",
                   "firewalld": "running not-running", "virtualization": "none kvm qemu vmware microsoft oracle xen uml parallels bhyve amazon zvm lxc lxc-libvirt docker podman systemd-nspawn wsl"}
        v = enum(text.strip(), allowed[probe])
        if v == "unknown":
            raise ValueError("unrecognized state")
        return {"state": v}
    if probe in {"firewall_zones", "firewall_rules"}:
        # Unknown/rich rules are fingerprinted, never exported verbatim. This is
        # an inspection aid; a reviewer must examine their local protected copy.
        lines = []
        for line in text.splitlines():
            stripped = line.strip()
            key, sep, value = stripped.partition(":")
            if sep and key in {"ports", "source-ports"}:
                ports = value.split()
                if not all(re.fullmatch(r"\d+(?:-\d+)?/(?:tcp|udp|sctp|dccp)", x) for x in ports):
                    raise ValueError("invalid firewall ports")
                lines.append({"field": key, "values": ports})
            elif sep and key == "services":
                known = {"ssh", "http", "https", "dhcp", "dhcpv6", "dhcpv6-client", "sip", "sips", "mysql", "ntp", "dns"}
                lines.append({"field": key, "values": [s if s in known else r.token(s) for s in value.split()]})
            elif sep and key in {"interfaces", "sources"}:
                lines.append({"field": key, "values": [r.token(s) for s in value.split()]})
            elif sep and key in {"forward", "masquerade", "icmp-block-inversion"}:
                lines.append({"field": key, "value": enum(value.strip(), "yes no")})
            elif stripped:
                lines.append({"rule_identity": r.token(stripped)})
        return {"configuration_lines": lines, "opaque_rules_require_local_review": True,
                "reachability_test": "NOT_RUN"}
    if probe == "time":
        p = pairs(text)
        require_fields(p, ("NTPSynchronized", "Timezone", "LocalRTC"))
        if not p["Timezone"].strip():
            raise ValueError("missing time properties")
        return {"ntp_synchronized": required_enum(p["NTPSynchronized"], "yes no"),
                "timezone_identity": r.token(p["Timezone"]), "local_rtc": required_enum(p["LocalRTC"], "yes no")}
    if probe == "chrony":
        wanted = {"Stratum", "System time", "Last offset", "RMS offset", "Leap status"}
        p = {}
        for line in text.splitlines():
            key, sep, value = line.partition(":")
            key = key.strip()
            if sep and key in wanted:
                if key == "Leap status":
                    if value.strip() not in {"Normal", "Insert second", "Delete second", "Not synchronised"}:
                        raise ValueError("invalid leap status")
                    p[key] = value.strip()
                else:
                    pattern = (r"\s*(\d+)\s*" if key == "Stratum" else
                               r"\s*([+-]?\d+(?:\.\d+)?) seconds(?: (slow|fast) of NTP time)?\s*")
                    match = re.fullmatch(pattern, value)
                    if not match:
                        raise ValueError("invalid chrony metric")
                    if key == "Stratum" and int(match[1]) > 16:
                        raise ValueError("invalid stratum")
                    if key == "System time":
                        if not match[2]:
                            raise ValueError("missing offset direction")
                        p["System time direction"] = match[2]
                    p[key] = match[1]
        if not wanted <= p.keys():
            raise ValueError("missing chrony tracking")
        return p
    if probe == "services":
        result = []
        for block in text.strip().split("\n\n"):
            p = pairs(block)
            require_fields(p, ("Id", "LoadState", "ActiveState", "SubState", "UnitFileState"))
            if p.get("Id") not in UNITS:
                raise ValueError("unknown service")
            result.append({"unit": p["Id"], "load": required_enum(p["LoadState"], "loaded not-found masked error bad-setting stub merged"),
                           "active": required_enum(p["ActiveState"], "active inactive failed activating deactivating reloading maintenance refreshing"),
                           "substate": required_enum(p["SubState"], "running dead exited failed condition start-pre start start-post auto-restart auto-restart-queued reload reload-signal reload-notify stop stop-watchdog stop-sigterm stop-sigkill stop-post final-watchdog final-sigterm final-sigkill cleaning"),
                           "enabled": "not_applicable" if p["LoadState"] == "not-found" and not p["UnitFileState"] else required_enum(p["UnitFileState"], "enabled disabled static masked indirect generated transient alias enabled-runtime linked linked-runtime masked-runtime bad")})
        if {x["unit"] for x in result} != set(UNITS):
            raise ValueError("incomplete service inventory")
        return result
    if probe == "listeners":
        result = []
        for line in text.splitlines():
            f = line.split()
            if len(f) < 6 or f[0] not in {"tcp", "udp"}:
                raise ValueError("invalid listener")
            host, port = f[4].rsplit(":", 1)
            if number(port) is None or number(port) > 65535:
                raise ValueError("invalid listener port")
            result.append({"protocol": f[0], "port": number(port),
                           "binding": {"wildcard": True} if host == "*" else address(host.strip("[]"), r)})
        return result
    if probe == "binaries":
        data = json.loads(text)
        if set(data) != set(BINARY_PATHS) or any(type(v) is not bool for v in data.values()):
            raise ValueError("invalid binary presence")
        return {"known_paths_present": data, "executed": False,
                "unmanaged_version_and_nonstandard_paths": "NOT_RUN"}
    raise ValueError("unknown probe")


def run_command(argv: tuple[str, ...], timeout: float = 10, limit: int = LIMIT) -> dict:
    """Bound both time and combined output; never export raw exception strings."""
    try:
        p = subprocess.Popen(argv, stdin=subprocess.DEVNULL, stdout=subprocess.PIPE,
                             stderr=subprocess.PIPE, env={"PATH": "/usr/sbin:/usr/bin:/sbin:/bin", "LC_ALL": "C", "LANG": "C", "SYSTEMD_PAGER": "cat", "SYSTEMD_COLORS": "0"})
    except FileNotFoundError:
        return {"error": "command_not_found", "exit_code": None}
    except OSError:
        return {"error": "command_unavailable", "exit_code": None}
    output = {"stdout": bytearray(), "stderr": bytearray()}
    error = None
    deadline = time.monotonic() + timeout
    with selectors.DefaultSelector() as sel:
        for name in output:
            sel.register(getattr(p, name), selectors.EVENT_READ, name)
        try:
            while sel.get_map():
                left = deadline - time.monotonic()
                if left <= 0:
                    error = "timeout"
                    break
                for key, _ in sel.select(min(left, 0.1)):
                    chunk = os.read(key.fileobj.fileno(), 8192)
                    if not chunk:
                        sel.unregister(key.fileobj)
                    elif sum(len(v) for v in output.values()) + len(chunk) > limit:
                        error = "output_limit"
                        break
                    else:
                        output[key.data].extend(chunk)
                if error:
                    break
            if error:
                p.kill()
            try:
                code = p.wait(timeout=max(0.01, deadline - time.monotonic()))
            except subprocess.TimeoutExpired:
                error = "timeout"
                p.kill()
                code = p.wait()
        finally:
            if p.poll() is None:
                p.kill()
                p.wait()
            p.stdout.close()
            p.stderr.close()
    return {**{k: bytes(v).decode("utf-8", "replace") for k, v in output.items()},
            "exit_code": code, "error": error}


def bounded_read(path: Path) -> str:
    with path.open("rb") as f:
        data = f.read(LIMIT + 1)
    if len(data) > LIMIT:
        raise ValueError("output_limit")
    return data.decode("utf-8", "strict")


def live_probe(probe: str) -> dict:
    if probe in COMMANDS:
        return run_command(COMMANDS[probe])
    try:
        if probe in FILES:
            value = bounded_read(Path(FILES[probe]))
        elif probe == "repositories":
            paths = sorted(Path("/etc/yum.repos.d").glob("*.repo"))
            if not paths or len(paths) > 128:
                raise ValueError("repository_config_unavailable")
            configs = [bounded_read(p) for p in paths]
            if sum(len(s.encode()) for s in configs) > LIMIT:
                raise ValueError("output_limit")
            value = json.dumps({"configs": configs, "dnf_config": bounded_read(Path("/etc/dnf/dnf.conf"))})
        else:
            value = json.dumps({p: Path(p).is_file() for p in BINARY_PATHS})
        return {"stdout": value, "exit_code": None, "file_read": True}
    except PermissionError:
        return {"error": "permission_denied", "exit_code": None}
    except FileNotFoundError:
        return {"error": "file_not_found", "exit_code": None}
    except (OSError, ValueError, UnicodeError):
        return {"error": "file_read_error", "exit_code": None}


def result(probe: str, raw: dict | None, r: Redactor, simulated: bool) -> dict:
    output = {"id": probe, "argv": list(COMMANDS.get(probe, ())),
              "method": "command" if probe in COMMANDS else "file_read",
              "observed_at_utc": utc(), "simulated": simulated, "exit_code": None,
              "status": "NOT_RUN", "facts": None, "error": None,
              "stderr_present": False, "stderr_bytes": 0}
    if raw is None:
        return output
    code = raw.get("exit_code")
    if code is not None and type(code) is not int:
        output.update(status="ERROR", error="invalid_exit_code")
        return output
    output.update(exit_code=code, stderr_present=bool(raw.get("stderr")),
                  stderr_bytes=len(str(raw.get("stderr", "")).encode()))
    error = raw.get("error")
    if error:
        output.update(status="UNAVAILABLE" if error in {"command_not_found", "command_unavailable", "file_not_found"} else "ERROR",
                      error=enum(error, "command_not_found command_unavailable file_not_found permission_denied file_read_error timeout output_limit"))
        return output
    text = raw.get("stdout", "")
    if not isinstance(text, str) or len(text.encode()) > LIMIT:
        output.update(status="ERROR", error="invalid_output")
        return output
    allowed_nonzero = (probe == "packages" and code == 1) or (probe == "virtualization" and code == 1 and text.strip() == "none") or (probe == "firewalld" and code == 252 and text.strip() == "not running")
    if code not in (None, 0) and not allowed_nonzero:
        output.update(status="ERROR", error="nonzero_exit")
        return output
    if probe in COMMANDS and code is None:
        output.update(status="ERROR", error="missing_exit_code")
        return output
    try:
        # firewalld's stopped result is a fact, not a successful rule query.
        if probe == "firewalld" and text.strip() == "not running":
            text = "not-running"
        facts = parse(probe, text, r)
        output.update(status="COLLECTED", facts=facts)
    except (ValueError, KeyError, TypeError, AttributeError, configparser.Error, RecursionError):
        output.update(status="ERROR", error="parse_error")
    return output


def inventory(fixture: dict | None = None, references: dict | None = None) -> dict:
    simulated = fixture is not None
    r = Redactor()
    started = utc()
    data = fixture.get("probes", {}) if simulated else None
    rows = []
    for probe in PROBES:
        raw = data.get(probe) if simulated else live_probe(probe)
        item = result(probe, raw, r, simulated)
        rows.append(item)
        if probe == "os" and item["status"] != "COLLECTED" and not simulated:
            # Fail closed before invoking any other host command.
            rows.extend(result(p, None, r, False) for p in PROBES[1:])
            break
    return {"schema_version": 1, "mode": "FIXTURE" if simulated else "HOST_COLLECTION",
            "collector_sha256": hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),
            "started_at_utc": started, "finished_at_utc": utc(),
            "references": {k: r.token(v) for k, v in (references or {}).items()},
            "sanitization": "allowlisted facts; per-run HMAC identities; no raw stdout/stderr or key exported",
            "host_baseline_acceptance": "NOT_RUN", "compatibility": "NOT_RUN",
            "independent_qa": "NOT_RUN", "probes": rows}


def summary(doc: dict) -> str:
    lines = ["# M0-01 host inventory", "", f"Mode: **{doc['mode']}**",
             f"UTC: {doc['started_at_utc']} to {doc['finished_at_utc']}",
             "", "Host baseline acceptance / compatibility / independent QA: **NOT_RUN**.",
             "Fixture values are synthetic and cannot establish host evidence.", "",
             "| Probe | Collection status | Exit code | Error |", "|---|---|---|---|"]
    for row in doc["probes"]:
        lines.append(f"| {row['id']} | {row['status']} | {row['exit_code']} | {row['error'] or ''} |")
    lines.extend(["", "## Sanitized facts", ""])
    for row in doc["probes"]:
        if row["facts"] is not None:
            lines.extend([f"### {row['id']}", "", "```json", json.dumps(row["facts"], indent=2, sort_keys=True), "```", ""])
    return "\n".join(lines) + "\n"


def write_outputs(directory: Path, doc: dict) -> None:
    # Require a new directory to avoid replacing evidence or following symlinks.
    directory.mkdir(mode=0o700)
    for name, content in (("inventory.json", json.dumps(doc, indent=2, sort_keys=True) + "\n"),
                          ("summary.md", summary(doc))):
        fd = os.open(directory / name, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
        with os.fdopen(fd, "w", encoding="utf-8") as f:
            f.write(content)


def main(argv: list[str] | None = None) -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    mode = ap.add_mutually_exclusive_group(required=True)
    mode.add_argument("--fixture", type=Path, help="offline replay; never falls back to host collection")
    mode.add_argument("--collect", action="store_true", help="local host only; requires separate exact-target authority")
    ap.add_argument("--host-ref", help="synthetic reference to separately recorded disposable host identity")
    ap.add_argument("--authorization-ref", help="reference to exact read-only host authorization")
    ap.add_argument("--snapshot-ref", help="reference to recoverable clean snapshot")
    ap.add_argument("--output-dir", type=Path, required=True, help="new evidence directory; existing paths are refused")
    args = ap.parse_args(argv)
    refs = {k: getattr(args, k) for k in ("host_ref", "authorization_ref", "snapshot_ref")}
    if args.collect and not all(refs.values()):
        ap.error("--collect requires --host-ref, --authorization-ref and --snapshot-ref; flags do not grant authority")
    if args.output_dir.exists() or args.output_dir.is_symlink():
        ap.error("output directory already exists; evidence will not be overwritten")
    try:
        fixture = json.loads(bounded_read(args.fixture)) if args.fixture else None
        if args.fixture and (not isinstance(fixture, dict) or type(fixture.get("fixture_schema")) is not int or fixture.get("fixture_schema") != 1 or not isinstance(fixture.get("probes"), dict)):
            raise ValueError("invalid fixture")
        if fixture is not None and any(k not in PROBES or not isinstance(v, dict) for k, v in fixture["probes"].items()):
            raise ValueError("invalid probe fixture")
        doc = inventory(fixture, {k: v for k, v in refs.items() if v})
        write_outputs(args.output_dir, doc)
    except (OSError, ValueError, TypeError, RecursionError):
        print("Inventory could not be prepared; check fixture schema, input size and output permissions. No raw diagnostics exported.", file=sys.stderr)
        return 2
    # Partial facts are saved but cannot be mistaken for a complete collection.
    return 0 if all(p["status"] == "COLLECTED" for p in doc["probes"]) else 1


if __name__ == "__main__":
    sys.exit(main())
