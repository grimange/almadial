// M0-05 diagnostic correction: capture both peer streams on one audio clock.
// ScriptProcessor callbacks dropped unequal frame counts in repeated ALM-5 runs.
// AudioWorklet runs on the audio rendering thread; analysis thresholds are unchanged.
// Load only in the authorized native Echo test page after both streams exist.
alm.capture = async seconds => {
  if (!alm.localStream || !alm.remoteStream) throw new Error('Both peer streams required');
  if (!Number.isFinite(seconds) || seconds < 1 || seconds > 30) throw new Error('Capture duration out of range');
  const ctx = new AudioContext({sampleRate: 16000});
  const url = URL.createObjectURL(new Blob([`
    class PairedCapture extends AudioWorkletProcessor {
      process(inputs, outputs) {
        const left = inputs[0]?.[0], right = inputs[1]?.[0];
        if (left && right) this.port.postMessage([left.slice(), right.slice()]);
        return true;
      }
    }
    registerProcessor('alm5-paired-capture', PairedCapture);
  `], {type: 'text/javascript'}));
  try {
    await ctx.audioWorklet.addModule(url);
    const node = new AudioWorkletNode(ctx, 'alm5-paired-capture', {numberOfInputs: 2, numberOfOutputs: 1, outputChannelCount: [1]});
    const local = ctx.createMediaStreamSource(alm.localStream), remote = ctx.createMediaStreamSource(alm.remoteStream);
    const chunks = [[], []];
    node.port.onmessage = e => {chunks[0].push(e.data[0]); chunks[1].push(e.data[1]);};
    local.connect(node, 0, 0); remote.connect(node, 0, 1); node.connect(ctx.destination);
    await ctx.resume(); const start = new Date().toISOString();
    await new Promise(resolve => setTimeout(resolve, seconds * 1000));
    local.disconnect(); remote.disconnect(); node.disconnect(); await ctx.close();
    // Flush message events queued before the worklet stopped.
    await new Promise(resolve => setTimeout(resolve, 50));
    function wav(parts) {
      const count = parts.reduce((n, x) => n + x.length, 0), buffer = new ArrayBuffer(44 + count * 2), view = new DataView(buffer);
      const text = (o, s) => [...s].forEach((c, i) => view.setUint8(o + i, c.charCodeAt(0)));
      text(0, 'RIFF'); view.setUint32(4, 36 + count * 2, true); text(8, 'WAVE'); text(12, 'fmt ');
      view.setUint32(16, 16, true); view.setUint16(20, 1, true); view.setUint16(22, 1, true);
      view.setUint32(24, 16000, true); view.setUint32(28, 32000, true); view.setUint16(32, 2, true); view.setUint16(34, 16, true);
      text(36, 'data'); view.setUint32(40, count * 2, true); let offset = 44;
      for (const part of parts) for (const sample of part) {view.setInt16(offset, Math.round(Math.max(-1, Math.min(1, sample)) * 32767), true); offset += 2;}
      return new Blob([buffer], {type: 'audio/wav'});
    }
    alm.captures = {local: wav(chunks[0]), remote: wav(chunks[1])};
    const result = {start, seconds, method: 'AudioWorklet paired inputs', localBytes: alm.captures.local.size, remoteBytes: alm.captures.remote.size};
    alm.events.push({type: 'capture', ...result}); return result;
  } finally {URL.revokeObjectURL(url); if (ctx.state !== 'closed') await ctx.close();}
};
