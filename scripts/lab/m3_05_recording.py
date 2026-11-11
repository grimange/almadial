#!/usr/bin/env python3
"""Content-only verification of the two-source synthetic MixMonitor recording.

Asterisk UNIQUEID timestamps channel creation, not the first recorded PCM frame.
Search the actual file for the simultaneous browser capture; report that offset
as content alignment, never as a measured recording-start time or network delay.
Call/file/database provenance must be verified separately.
"""
import math
from pathlib import Path
import m0_03_signal as signal


def analyze(recording, source_a, source_b):
    a = signal.features(signal.read_pcm(Path(source_a)))
    b = signal.features(signal.read_pcm(Path(source_b)))
    pcm = signal.read_pcm(Path(recording))
    size = int(signal.RATE * signal.FRAME_SECONDS)
    powers = []
    for start in range(0, len(pcm) - size + 1, size):
        row = []
        for frequency in signal.FREQUENCIES:
            coefficient = 2 * math.cos(2 * math.pi * frequency / signal.RATE)
            x = y = 0.
            for sample in pcm[start:start + size]:
                z = sample + coefficient*x - y
                y, x = x, z
            row.append(max(0., x*x+y*y-coefficient*x*y))
        powers.append(row)
    best = {'agreement': 0., 'frames': 0, 'offset_seconds': None, 'tones': []}
    for offset in range(max(0, len(powers) - min(len(a), len(b)) + 1)):
        matches = total = 0
        seen = set()
        for i, (left, right) in enumerate(zip(a, b)):
            if i < 50 or min(left[0], right[0]) < .02 or left[1] == right[1] or min(left[1], right[1]) < 0:
                continue
            expected = {left[1], right[1]}
            row = powers[offset + i]
            total += 1
            if set(sorted(range(4), key=lambda k: row[k])[-2:]) == expected and min(row[k] for k in expected) >= .08 * sum(row):
                matches += 1
                seen.update(expected)
        score = matches / max(1, total)
        if score > best['agreement']:
            best = {'agreement': score, 'frames': total, 'offset_seconds': offset * signal.FRAME_SECONDS,
                    'tones': sorted(signal.FREQUENCIES[i] for i in seen)}
    best.update(analysis_state='PASS' if best['agreement'] >= .90 and best['frames'] >= 200 and len(best['tones']) == 4 else 'FAIL',
                runtime_state='NOT_RUN', method='whole-file content alignment; independent call provenance required',
                thresholds={'agreement': .90, 'distinguishable_frames': 200, 'minimum_each_tone_fraction': .08})
    return best
