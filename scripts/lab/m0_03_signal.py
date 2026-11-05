#!/usr/bin/env python3
"""Generate/analyze the synthetic ALM-3 browser-media fixture (no host effects)."""
import argparse
import array
import hashlib
import json
import math
import pathlib
import statistics
import sys
import wave

RATE = 16000
FREQUENCIES = (697, 941, 1209, 1633)
PATTERN = (0, 2, 1, 3, 1, 0, 3, 2, 0, 3, 2, 1, 2, 3, 0, 1)
FRAME_SECONDS = 0.02


def fixture(path, seconds=40):
    samples = array.array('h')
    for i in range(RATE * seconds):
        t = i / RATE
        slot = int(t / 0.5) % len(PATTERN)
        phase = t % 0.5
        # 400 ms tone and 100 ms silence; short ramp avoids edge clicks.
        gain = min(1, phase / 0.01, max(0, (0.4 - phase) / 0.01))
        amplitude = 0.22 if slot % 3 else 0.12
        samples.append(round(32767 * amplitude * gain * math.sin(2 * math.pi * FREQUENCIES[PATTERN[slot]] * t)))
    if sys.byteorder != 'little':
        samples.byteswap()
    with wave.open(str(path), 'wb') as out:
        out.setparams((1, 2, RATE, 0, 'NONE', 'not compressed'))
        out.writeframes(samples.tobytes())
    return hashlib.sha256(pathlib.Path(path).read_bytes()).hexdigest()


def read_pcm(path):
    with wave.open(str(path), 'rb') as src:
        if (src.getnchannels(), src.getsampwidth(), src.getframerate()) != (1, 2, RATE):
            raise ValueError('Expected mono signed PCM16 WAV at 16000 Hz')
        samples = array.array('h', src.readframes(src.getnframes()))
    if sys.byteorder != 'little':
        samples.byteswap()
    return [x / 32768 for x in samples]


def features(samples):
    size = int(RATE * FRAME_SECONDS)
    result = []
    for start in range(0, len(samples) - size + 1, size):
        frame = samples[start:start + size]
        rms = math.sqrt(sum(x*x for x in frame) / size)
        powers = []
        for freq in FREQUENCIES:
            coeff = 2 * math.cos(2 * math.pi * freq / RATE)
            a = b = 0.0
            for x in frame:
                c = x + coeff*a - b
                b, a = a, c
            powers.append(max(0, a*a+b*b-coeff*a*b))
        label = max(range(4), key=lambda x: powers[x]) if rms >= 0.015 else -1
        result.append((rms, label))
    return result


def correlation(a, b):
    ma, mb = statistics.mean(a), statistics.mean(b)
    da, db = [x-ma for x in a], [x-mb for x in b]
    denom = math.sqrt(sum(x*x for x in da)*sum(x*x for x in db))
    return sum(x*y for x, y in zip(da, db))/denom if denom else 0.0


def compare(local_path, remote_path):
    local, remote = features(read_pcm(local_path)), features(read_pcm(remote_path))
    if min(len(local), len(remote)) < 400:
        raise ValueError('At least eight seconds of simultaneous capture required')
    # Ignore startup; both streams must use the same AudioContext clock.
    best = None
    for lag in range(101):
        end = min(len(local), len(remote)-lag)
        left, right = local[50:end], remote[50+lag:end+lag]
        score = correlation([x[0] for x in left], [x[0] for x in right])
        if best is None or score > best[0]:
            best = (score, lag, left, right)
    score, lag, left, right = best
    tones = [(a,b) for a,b in zip(left,right) if a[1] >= 0]
    agreement = sum(a[1] == b[1] for a,b in tones) / max(1,len(tones))
    # Match the actual injected input against every possible fixture phase.
    input_labels = [x[1] for x in local]
    pattern_score = 0
    for phase in range(400):
        matches = total = 0
        for i, label in enumerate(input_labels):
            k = (i + phase) % 400
            # Exclude transition frames where resampling can straddle a boundary.
            if k % 25 in (0,19,20,24):
                continue
            expected = PATTERN[k // 25] if k % 25 < 20 else -1
            matches += label == expected
            total += 1
        pattern_score = max(pattern_score, matches / max(1,total))
    seen = sorted({x[1] for x in right if x[1] >= 0})
    remote_rms = math.sqrt(statistics.mean(x[0]**2 for x in right))
    passed = score >= .85 and agreement >= .90 and pattern_score >= .90 and len(seen)==4 and remote_rms >= .02 and lag < 100
    return {'state':'PASS' if passed else 'FAIL','fixture_pattern_match':pattern_score,
            'envelope_correlation':score,'tone_label_agreement':agreement,'estimated_latency_ms':lag*20,
            'remote_rms':remote_rms,'frequencies_observed_hz':[FREQUENCIES[x] for x in seen],
            'local_frames':len(local),'remote_frames':len(remote),
            'thresholds':{'pattern_match':.90,'envelope_correlation':.85,'tone_agreement':.90,'remote_rms':.02,'max_latency_ms_exclusive':2000},
            'physical_microphone_speaker':'NOT_RUN', 'note':'Signal result must also be joined with real browser ICE/DTLS/RTP and Asterisk Echo evidence.'}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest='mode', required=True)
    gen = sub.add_parser('generate'); gen.add_argument('path',type=pathlib.Path)
    check = sub.add_parser('compare'); check.add_argument('local',type=pathlib.Path); check.add_argument('remote',type=pathlib.Path)
    args = parser.parse_args()
    if args.mode == 'generate':
        if args.path.exists():
            parser.error('Refusing to overwrite fixture')
        print(json.dumps({'sha256':fixture(args.path),'sample_rate':RATE,'pattern':PATTERN,'frequencies_hz':FREQUENCIES}))
    else:
        result = compare(args.local,args.remote); print(json.dumps(result,indent=2)); return result['state']!='PASS'
    return 0

if __name__ == '__main__':
    raise SystemExit(main())
