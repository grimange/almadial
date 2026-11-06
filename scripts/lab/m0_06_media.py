#!/usr/bin/env python3
"""Offline analysis of actual ALM-6 cross-participant recordings, never runtime PASS.

Paired AudioWorklets within each browser share a clock. Independent browser clocks
must start within the recorded capture skew, reported separately. Quality thresholds
are inherited unchanged from ALM-3. Call/bridge provenance must be reviewed separately.
"""
import argparse
import json
import math
from pathlib import Path
import m0_03_signal as signal

PATTERNS = {
    '9051': (0, 2, 1, 3, 1, 0, 3, 2, 0, 3, 2, 1, 2, 3, 0, 1),
    '9052': (3, 1, 2, 0, 2, 3, 0, 1, 3, 0, 1, 2, 1, 0, 3, 2),
    '9053': (0, 1, 2, 3, 0, 2, 3, 1, 1, 3, 0, 2, 2, 0, 1, 3),
}


def compare(source, remote, identity):
    """Only the expected injected pattern changes; never lower media thresholds."""
    old = signal.PATTERN
    try:
        signal.PATTERN = PATTERNS[identity]
        result = signal.compare(source, remote)
    finally:
        signal.PATTERN = old
    result['analysis_state'] = result.pop('state')
    result['runtime_state'] = 'NOT_RUN'
    result['source_identity'] = identity
    result['note'] = ('Offline signal evidence only. Join with trusted browser SDP/DTLS, '
                      'independent remote-track capture and actual ConfBridge channel events. '
                      'Reported delay includes independent capture-start skew.')
    return result


def two_party(root, prefix):
    root = Path(root)
    result = {}
    for source, destination in [('9051', '9052'), ('9052', '9051')]:
        result[source + '-to-' + destination] = compare(
            root / (prefix + '-' + source + '-local.wav'),
            root / (prefix + '-' + destination + '-remote.wav'), source)
    return result


def mixed_recording(recording, local_a, local_b, offset_seconds):
    """Verify both distinct source tones in an actual decoded conference recording.

    offset_seconds comes from the recorded capture UTC minus AMI record-start UTC.
    Search only +/-200 ms around it, allowing codec/clock alignment. Ignore frames
    where the sources have the same tone or are silent; those cannot prove mixing.
    This is content analysis only, never a claim that a recording control path ran.
    """
    a, b = signal.features(signal.read_pcm(local_a)), signal.features(signal.read_pcm(local_b))
    pcm = signal.read_pcm(recording)
    size = int(signal.RATE * signal.FRAME_SECONDS)
    powers = []
    for start in range(0, len(pcm) - size + 1, size):
        frame = pcm[start:start + size]
        row = []
        for frequency in signal.FREQUENCIES:
            coeff = 2 * math.cos(2 * math.pi * frequency / signal.RATE)
            x = y = 0.
            for sample in frame:
                z = sample + coeff*x - y
                y, x = x, z
            row.append(max(0., x*x + y*y - coeff*x*y))
        powers.append(row)
    origin = round(offset_seconds / signal.FRAME_SECONDS)
    best = (0., 0, 0, set())
    for offset in range(origin - 10, origin + 11):
        matches = total = 0
        seen = set()
        for i, (left, right) in enumerate(zip(a, b)):
            if i < 50 or min(left[0], right[0]) < .02 or left[1] == right[1]:
                continue
            if not 0 <= offset + i < len(powers):
                continue
            expected = {left[1], right[1]}
            if -1 in expected:
                continue
            row = powers[offset + i]
            total += 1
            strongest = set(sorted(range(4), key=lambda k: row[k])[-2:])
            if strongest == expected and min(row[k] for k in expected) >= .08 * sum(row):
                matches += 1
                seen.update(expected)
        score = matches / max(1, total)
        if score > best[0]:
            best = (score, total, offset, seen)
    score, total, offset, seen = best
    return {'analysis_state': 'PASS' if score >= .90 and total >= 200 and len(seen) == 4 else 'FAIL',
            'runtime_state': 'NOT_RUN', 'both_source_tones_agreement': score,
            'distinguishable_frames': total, 'alignment_seconds': offset * signal.FRAME_SECONDS,
            'thresholds': {'both_source_tones_agreement': .90, 'minimum_distinguishable_frames': 200,
                           'minimum_each_tone_fraction': .08, 'alignment_search_ms': 200},
            'note': 'Decoded recording content only; correlate file, AMI record events and application row separately.'}


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('evidence', type=Path)
    parser.add_argument('prefix')
    args = parser.parse_args()
    results = two_party(args.evidence, args.prefix)
    print(json.dumps(results, indent=2))
    raise SystemExit(any(r['analysis_state'] != 'PASS' for r in results.values()))
