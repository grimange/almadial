#!/usr/bin/env python3
"""Offline ALM-32 coverage and source preparation; never publishes or installs.

Reads only pinned regular files. The resulting local archive is a review
candidate, not a representation that redistribution rights have been cleared.
"""
import argparse
import difflib
import io
import json
import os
from pathlib import Path, PurePosixPath
import re
import tarfile
import zipfile

from generate_sbom import DEFAULT, ROOT, digest, encoded, read_inputs


def regular(path):
    if path.is_symlink() or not path.is_file():
        raise ValueError('expected regular file: ' + str(path))
    return path.read_bytes()


def safe(name):
    p = PurePosixPath(name)
    if p.is_absolute() or '..' in p.parts or str(p) != name:
        raise ValueError('unsafe member: ' + name)
    return name


def archive_files(path, expected):
    raw = regular(path)
    if digest(raw) != expected['sha256']:
        raise ValueError('archive checksum mismatch: ' + path.name)
    files = {}
    if path.suffix == '.zip':
        with zipfile.ZipFile(io.BytesIO(raw)) as z:
            for item in z.infolist():
                if not item.is_dir():
                    if (item.external_attr >> 16) & 0o170000 == 0o120000:
                        raise ValueError('archive symlink')
                    if item.filename in files:
                        raise ValueError('duplicate archive member')
                    files[safe(item.filename)] = z.read(item)
    else:
        with tarfile.open(fileobj=io.BytesIO(raw)) as t:
            for item in t:
                # Original upstream link entries are retained inside the intact
                # archive, never extracted or followed by this utility.
                if item.isfile():
                    if item.name in files:
                        raise ValueError('duplicate archive member')
                    files[safe(item.name)] = t.extractfile(item).read()
    actual = [{'path': n, 'bytes': len(b), 'sha256': digest(b)}
              for n, b in sorted(files.items())]
    if actual != sorted(expected['files'], key=lambda x: x['path']):
        raise ValueError('source file inventory mismatch: ' + path.name)
    return raw, files


def coverage(base=DEFAULT):
    v = read_inputs(base)
    headers = v['source-headers.json']
    rows = []
    for c in v['components.json']:
        evidence = c['evidence']
        members = []
        if c['sha256']:
            members = [dict(archive=a, path=f['path'])
                       for a, spec in v['source-files.json'].items()
                       for f in spec['files'] if f['sha256'] == c['sha256']]
        if c['category'] == 'downloaded-source':
            members = [dict(archive=a, path='* (membership in source-files.json)')
                       for a, spec in v['source-files.json'].items()
                       if spec['sha256'] == c['sha256']]
        text = (base / evidence).read_text(errors='replace')
        if evidence == 'source-headers.json':
            selected = [h for h in headers if any(h['archive'] == f['archive'] and
                        h['path'] == f['path'] for f in members)]
            text = '\n'.join('\n'.join(h['header']) for h in selected)
        copyright_lines = [s.strip() for s in text.splitlines()
                           if re.search(r'copyright|©|\(c\)', s, re.I)]
        exception = []
        if c['license'] in ('NOASSERTION', 'LicenseRef-LineJS-Conflicting'):
            exception.append('Unresolved license terms; no distribution clearance')
        if not copyright_lines:
            exception.append('Copyright not concluded; consult preserved source notices')
        if c['category'] in ('downloaded-source', 'embedded-source-notice'):
            exception.append('Source membership known; exact linked build closure not established')
        rows.append(dict(name=c['name'], version=c['version'], source=c['url'],
            parent=c['parent'], delivered_files=members or ['category membership in repository/module inventory'],
            license=c['license'], license_evidence=evidence,
            copyright_evidence=copyright_lines or ['NOASSERTION'],
            modification_status=('See source-delivery-manifest.json' if c['name'] in ('VICIdial', 'VICIphone')
                                 else 'No installer transformation identified; source membership does not imply linked inclusion'),
            distribution_category=c['category'], notice='docs/THIRD_PARTY_NOTICES.md plus exact evidence file',
            source_obligation='Review exact license and actual conveyance/network use; see distribution-model.json',
            classification='DOCUMENTED_EXCEPTION' if exception else 'RESOLVED', exceptions=exception))
    mapped = {(e.get('source_archive'), e.get('source_path')) for e in v['license-evidence.json']}
    candidates = [dict(archive=a, path=f['path'], sha256=f['sha256'])
                  for a, spec in v['source-files.json'].items() for f in spec['files']
                  if re.match(r'^(license|copying|copyright|notice)([._-].*)?$',
                              PurePosixPath(f['path']).name, re.I)
                  and (a, f['path']) not in mapped]
    return dict(scope='Existing 40 curated records, not a claim of complete per-file legal clearance',
        total_components_evaluated=len(rows), fully_mapped=sum(not r['exceptions'] for r in rows),
        components_with_missing_license_evidence=0,
        components_with_unconcluded_license_declarations=sum(r['license'] == 'NOASSERTION' for r in rows),
        components_with_conflicting_evidence=sum(r['license'] == 'LicenseRef-LineJS-Conflicting' for r in rows),
        components_with_explicit_exceptions=sum(bool(r['exceptions']) for r in rows),
        components_with_missing_notice_coverage=0,
        notice_scope='Every curated row references included evidence; remaining candidates below are NOT cleared',
        components_not_in_installer_runtime=sum(r['distribution_category'] in ('test-only','validation-data') for r in rows),
        embedded_md5_portions=[dict(name='TypeScript MD5 / Raymond Hill portion',
            version='Exact SIP.js 0.20.1 asset SHA-256 in source-files.json', license='MIT',
            copyright='Copyright (C) 2014 Raymond Hill', modified=False,
            delivered_files='viciphone.tar.gz:src/js/sip.js, MD5 comment at lines 11330-11386',
            evidence='licenses/SIP.js-0.20.1/embedded-notices.txt',
            distribution='Browser-served source; include full MIT notice; no copyleft source offer'),
            dict(name='TypeScript MD5 / Andre Cruz portion',
            version='Exact SIP.js 0.20.1 asset SHA-256 in source-files.json', license='WTFPL',
            copyright='Copyright (C) 2015 André Cruz', modified=False,
            delivered_files='viciphone.tar.gz:src/js/sip.js, same complete MD5 comment',
            evidence='licenses/SIP.js-0.20.1/embedded-notices.txt',
            distribution='Browser-served source; full original notice preserved; no copyleft source offer')],
        unreviewed_notice_candidates=candidates, components=rows)


def modifications(files, root=ROOT):
    """Read exact transformations from hash-bound accepted Go source, no installer execution."""
    inventory = json.loads((DEFAULT / 'patch-and-build-inventory.json').read_bytes())
    code = {}
    for item in inventory:
        if item['path'].endswith('_source.go'):
            data = regular(root / item['path'])
            if digest(data) != item['sha256']:
                raise ValueError('transformation source changed: ' + item['path'])
            code[item['path']] = data.decode()
    changes = {}
    v = code['internal/provision/vicidial_source.go']
    pins = re.findall(r'"(bin/[^"\n]+)":\s*"([0-9a-f]{64})"', v)
    literals = re.findall(r'`(/bin/ps[^`]+)`', v)
    if len(pins) != 4 or len(literals) != 2:
        raise ValueError('unsupported procps transform')
    for path, sha in pins:
        name = '2026-11-05/' + path
        raw = files['vicidial-r4037.zip'][name]
        if digest(raw) != sha or raw.count(literals[0].encode()) != (3 if 'keepalive' in path else 1):
            raise ValueError('procps source mismatch')
        changes[('vicidial-r4037.zip', name)] = raw.replace(literals[0].encode(), literals[1].encode())
    p = code['internal/provision/phone_source.go']
    section = p.split('replacements :=', 1)[1].split('\n\tfor name, changes', 1)[0]
    string_literal = r'"(?:[^"\\]|\\.)*"'
    blocks = re.split(r'\n\t\t("(?:viciphone.php|js/vici_phone.js)"): ', section)
    if len(blocks) != 5:
        raise ValueError('unsupported phone transform')
    for index in (1, 3):
        name = 'ViciPhone-24e98fccfa7b7079fff970e02fca5f002ac1f2cc/src/' + json.loads(blocks[index])
        raw = files['viciphone.tar.gz'][name]
        pairs = re.findall(r'\{(' + string_literal + r'), (' + string_literal + r')\}', blocks[index+1])
        if len(pairs) != (1 if index == 1 else 6):
            raise ValueError('phone transformation count mismatch')
        for old, new in pairs:
            before, after = json.loads(old).encode(), json.loads(new).encode()
            if raw.count(before) != 1:
                raise ValueError('phone source mismatch')
            raw = raw.replace(before, after, 1)
        changes[('viciphone.tar.gz', name)] = raw
    return changes


def prepare(archives, output, base=DEFAULT):
    specs = json.loads((base / 'source-files.json').read_bytes())
    members, files = {}, {}
    for name, spec in specs.items():
        raw, files[name] = archive_files(archives / name, spec)
        members['upstream/' + name] = raw
    changes = modifications(files)
    records = []
    for (archive, name), new in sorted(changes.items()):
        old = files[archive][name]
        path = 'modified/' + archive + '/' + name
        members[path] = new
        patch = ''.join(difflib.unified_diff(old.decode().splitlines(True), new.decode().splitlines(True),
                        fromfile='a/' + name, tofile='b/' + name)).encode()
        members['patches/' + archive + '/' + name + '.patch'] = patch
        records.append(dict(archive=archive, path=name, original_sha256=digest(old),
                            modified_sha256=digest(new), modification_notice='AlmaDial accepted installer transformations; qualification 2026-11-13'))
    # All tracked build/install Go code and embedded data, not unrelated operations evidence.
    repo = json.loads((base / 'repository-files.json').read_bytes())
    for item in repo:
        name = item['path']
        if name.startswith(('cmd/', 'internal/', 'manifests/', 'third_party/')) or name in (
                'go.mod', 'go.sum', 'LICENSE', 'scripts/build_go.py',
                'docs/implementation/m4-01-clean-host-installation-and-release-qualification.md',
                'docs/operations/almadial-operator-handover.md') or name.endswith('.patch'):
            raw = regular(ROOT / name)
            if digest(raw) != item['sha256']:
                raise ValueError('installer source mismatch: ' + name)
            members['installer/' + name] = raw
    for path in sorted(base.rglob('*')):
        if path.is_file():
            members['installer/docs/compliance/m4-04/' + str(path.relative_to(base))] = regular(path)
    members['notices/THIRD_PARTY_NOTICES.md'] = regular(ROOT / 'docs/THIRD_PARTY_NOTICES.md')
    members['SOURCE-DELIVERY.json'] = encoded(dict(classification='LOCAL_REVIEW_CANDIDATE_NOT_CLEARED_FOR_PUBLICATION',
        original_archives={n:s['sha256'] for n,s in specs.items()},
        modified_files=records, preparation_script_sha256=digest(regular(Path(__file__))),
        exceptions=['line.js unresolved rights', 'Nested notice candidates and linked build closure not cleared',
                    'No deployed network source-access mechanism; local candidate is not a source offer'],
        reconstruction='Unpack intact upstream archive, then overlay corresponding modified/<archive>/ tree. '
                       'Patches and installer Go source define exact accepted transformations. Build with installer/scripts/build_go.py.'))
    members['installer/scripts/prepare_compliance_source.py'] = regular(Path(__file__))
    members['installer/scripts/generate_sbom.py'] = regular(ROOT / 'scripts/generate_sbom.py')
    manifest = {n:digest(b) for n,b in sorted(members.items())}
    members['SHA256SUMS'] = ''.join(f'{sha}  {n}\n' for n,sha in manifest.items()).encode()
    # Exclusive creation prevents replacing previous source/evidence artifacts.
    with os.fdopen(os.open(output, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600), 'wb') as f, tarfile.open(fileobj=f, mode='w', format=tarfile.PAX_FORMAT) as t:
        for name, raw in sorted(members.items()):
            safe(name)
            h = tarfile.TarInfo(name)
            h.size, h.mode, h.mtime = len(raw), 0o600, 0
            t.addfile(h, io.BytesIO(raw))
    output.chmod(0o600)
    with tarfile.open(output) as t:
        for name, sha in manifest.items():
            if digest(t.extractfile(name).read()) != sha:
                raise ValueError('generated member checksum mismatch')
    return dict(archive_sha256=digest(output.read_bytes()), members=len(members), modified_files=records,
                classification='LOCAL_REVIEW_CANDIDATE_NOT_CLEARED_FOR_PUBLICATION')


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--check', action='store_true')
    p.add_argument('--archives', type=Path)
    p.add_argument('--output', type=Path)
    args = p.parse_args()
    path = DEFAULT / 'component-coverage.json'
    data = encoded(coverage())
    if args.check:
        if regular(path) != data:
            raise SystemExit('coverage differs from pinned inputs')
    else:
        path.write_bytes(data)
    if args.archives or args.output:
        if not (args.archives and args.output):
            p.error('--archives and --output must be supplied together')
        print(encoded(prepare(args.archives, args.output)).decode(), end='')


if __name__ == '__main__':
    main()
