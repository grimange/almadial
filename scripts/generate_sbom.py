#!/usr/bin/env python3
"""Generate the ALM-32 SPDX snapshot offline from reviewed, versioned inputs.

This records evidence, including unresolved licenses; it does not decide legality.
"""
import argparse
import hashlib
import json
from pathlib import Path
from urllib.parse import quote

ROOT = Path(__file__).resolve().parents[1]
DEFAULT = ROOT / 'docs/compliance/m4-04'


def digest(data):
    return hashlib.sha256(data).hexdigest()


def encoded(value):
    return (json.dumps(value, indent=2, sort_keys=True, ensure_ascii=True) + '\n').encode()


def read_inputs(base):
    names = ['baseline.json', 'components.json', 'rpm-inventory.json',
             'go-modules.json', 'source-files.json', 'source-headers.json',
             'license-evidence.json', 'repository-files.json', 'patch-and-build-inventory.json']
    values = {name: json.loads((base / name).read_bytes()) for name in names}
    for item in values['license-evidence.json']:
        if digest((base / item['path']).read_bytes()) != item['sha256']:
            raise ValueError('license evidence checksum mismatch: ' + item['path'])
    for component in values['components.json']:
        if not (base / component['evidence']).is_file():
            raise ValueError('missing license evidence: ' + component['name'])
    return values


def generate(base):
    v = read_inputs(base)
    baseline = v['baseline.json']
    packages = []
    relationships = []
    ids = {}
    def add(name, version, **fields):
        identifier = 'SPDXRef-' + digest((name + '\0' + version).encode())[:24]
        if identifier in ids.values():
            raise ValueError('duplicate package identity: ' + name)
        ids[name] = identifier
        package = dict(SPDXID=identifier, name=name, versionInfo=version,
                       downloadLocation='NOASSERTION', filesAnalyzed=False,
                       licenseConcluded='NOASSERTION', licenseDeclared='NOASSERTION',
                       copyrightText='NOASSERTION')
        package.update(fields)
        packages.append(package)
        return identifier
    root = add('AlmaDial qualification profile', baseline['git'],
               comment='Installed-system and source evidence snapshot; not a release payload. '
                       'Host RPMs are not asserted to be redistributed. Supplemental source-files.json '
                       'and repository-files.json contain per-file digests. Original-code MIT authority '
                       'is limited to the attestation in baseline.json.')
    for c in sorted(v['components.json'], key=lambda c: (c['name'], c['version'])):
        fields = dict(downloadLocation=c['url'] or 'NOASSERTION', licenseDeclared=c['license'],
                      licenseComments='Exact evidence: ' + c['evidence'] + '. ' + c['comments'],
                      comment='Inventory category: ' + c['category'])
        if c['sha256'] and c['category'] != 'embedded-source-notice':
            fields['checksums'] = [dict(algorithm='SHA256', checksumValue=c['sha256'])]
        identifier = add(c['name'], c['version'], **fields)
        relation = {'binary-linked': 'DEPENDS_ON', 'test-only': 'TEST_DEPENDENCY_OF',
                    'toolchain-vendored': 'OTHER', 'embedded-source-notice': 'OTHER'}.get(c['category'], 'CONTAINS')
        relationships.append(dict(spdxElementId=identifier if relation == 'TEST_DEPENDENCY_OF' else root,
                                  relationshipType=relation,
                                  relatedSpdxElement=root if relation == 'TEST_DEPENDENCY_OF' else identifier,
                                  comment='Evidence category: ' + c['category'] + '; no inferred build inclusion.'))
    for rpm in sorted(v['rpm-inventory.json'], key=lambda r: (r['name'], r['epoch'], r['version'], r['release'], r['arch'])):
        version = rpm['epoch'] + ':' + rpm['version'] + '-' + rpm['release']
        identifier = add('rpm:' + rpm['name'] + '.' + rpm['arch'], version,
            supplier='Organization: ' + rpm['vendor'] if rpm['vendor'] else 'NOASSERTION',
            licenseComments='Verbatim RPM license metadata (not a source-backed SPDX conclusion): ' + rpm['license_metadata'],
            sourceInfo='Source RPM: ' + rpm['source_rpm'],
            comment='Installed OS inventory; installer_selected=' + str(rpm['installer_selected']).lower() +
                    '; installed repository=' + str(rpm['installed_repository']) + '; archive checksum unavailable.',
            externalRefs=[dict(referenceCategory='PACKAGE-MANAGER', referenceType='purl',
                referenceLocator='pkg:rpm/almalinux/' + quote(rpm['name'], safe='') + '@' + quote(version, safe='') + '?arch=' + rpm['arch'])])
        relationships.append(dict(spdxElementId=root, relationshipType='CONTAINS', relatedSpdxElement=identifier,
                                  comment='Observed installed host; not a redistributed installer dependency claim.'))
    extracted = []
    for license_id in sorted({c['license'] for c in v['components.json'] if c['license'].startswith('LicenseRef-')}):
        c = next(c for c in v['components.json'] if c['license'] == license_id)
        path = base / c['evidence']
        if path.suffix == '.json':
            headers = json.loads(path.read_bytes())
            # Conflicting line.js notice is deliberately preserved as evidence, not normalized to GPL.
            candidates = ['\n'.join(item['header']) for item in headers if item['path'].endswith('/line.js')]
            if not candidates:
                raise ValueError('missing conflicting license header')
            text = candidates[0].split('*/', 1)[0] + '*/'
        else:
            raw = path.read_bytes()
            try:
                text = raw.decode('utf-8')
            except UnicodeDecodeError:
                text = raw.decode('latin-1')
        extracted.append(dict(licenseId=license_id, extractedText=text, name=license_id,
                              comment='Verbatim license evidence; original bytes retained at ' + c['evidence']))
    identity = digest(encoded(v))
    return dict(spdxVersion='SPDX-2.3', dataLicense='CC0-1.0', SPDXID='SPDXRef-DOCUMENT',
        name='AlmaDial ALM-32 installed-system and source inventory',
        documentNamespace='https://github.com/grimange/almadial/spdx/alm32-' + identity,
        creationInfo=dict(created=baseline['created'], creators=['Tool: almadial-generate-sbom-1'],
                          comment='Fixed qualification snapshot date, not wall-clock regeneration time.'),
        documentDescribes=[root], packages=sorted(packages, key=lambda p:p['SPDXID']),
        relationships=sorted(relationships, key=lambda r:(r['spdxElementId'],r['relationshipType'],r['relatedSpdxElement'])),
        hasExtractedLicensingInfos=extracted)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--inputs', type=Path, default=DEFAULT)
    parser.add_argument('--output', type=Path, default=DEFAULT / 'almadial.spdx.json')
    parser.add_argument('--check', action='store_true')
    args = parser.parse_args()
    data = encoded(generate(args.inputs))
    if args.check:
        if args.output.read_bytes() != data:
            raise SystemExit('SBOM differs from reviewed inputs')
    else:
        args.output.write_bytes(data)
    print('SPDX SHA256 ' + digest(data))


if __name__ == '__main__':
    main()
