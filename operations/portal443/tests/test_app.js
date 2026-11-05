'use strict';
// Deterministic countdown tests: mocked status transport and clock, real app.js.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const source = fs.readFileSync(path.join(__dirname, '..', 'app.js'), 'utf8');
const active = {state: 'active', remaining: 43199, destination: 'https://untrusted.invalid/'};
const flush = () => new Promise(resolve => setImmediate(resolve));

async function harness(statuses, hasContinue = true) {
  const elements = Object.fromEntries(['state', 'countdown', 'lifetime', 'continue'].map(id =>
    [id, {textContent: '', hidden: id === 'countdown', remove() { delete elements[id]; }}]));
  if (!hasContinue) delete elements.continue;
  const timers = [], navigation = [], requests = [];
  const context = {
    document: {getElementById: id => elements[id]},
    location: {assign: url => navigation.push(['assign', url]), replace: url => navigation.push(['replace', url])},
    setTimeout: callback => timers.push(callback),
    fetch: async (url, options) => {
      requests.push([url, options]);
      const status = statuses.shift();
      if (status instanceof Error) throw status;
      assert.ok(status, 'Test must supply every polled status');
      return {ok: true, json: async () => status};
    },
  };
  vm.runInNewContext(source, context);
  await flush();
  return {elements, navigation, requests, timers, async step() {
    assert.ok(timers.length, 'Expected a scheduled tick');
    timers.shift()();
    await flush();
  }};
}

(async () => {
  let h = await harness([{state: 'pending'}, active], false);
  assert.equal(h.elements.countdown.hidden, true);
  assert.equal(h.navigation.length, 0);
  await h.step();
  assert.deepEqual(h.navigation, [['replace', '/portal/']]);
  console.log('PASS pending does not start countdown; confirmation reloads server-rendered success');

  h = await harness(Array.from({length: 6}, () => ({...active})));
  assert.equal(h.elements.countdown.hidden, false);
  assert.equal(h.elements.countdown.textContent, 'Redirecting in: 5 seconds');
  assert.equal(h.elements.lifetime.textContent, '11h 59m');
  for (let i = 0; i < 4; i++) { await h.step(); assert.equal(h.navigation.length, 0); }
  await h.step();
  assert.deepEqual(h.navigation, [['assign', '/portal/continue']]);
  assert.equal(h.requests.length, 6);
  assert.ok(h.requests.every(([url, options]) => url === '/portal/status' && options.cache === 'no-store' && options.credentials === 'same-origin'));
  console.log('PASS five checked seconds; fixed continuation ignores supplied destination');

  for (const state of ['failed', 'expired', 'revoked']) {
    h = await harness([active, {state, remaining: 0, message: 'Access unavailable'}]);
    await h.step();
    assert.equal(h.elements.continue, undefined);
    assert.equal(h.navigation.length, 0);
    assert.equal(h.timers.length, 0);
    console.log(`PASS ${state} halts countdown and removes continuation`);
  }
  h = await harness([{...active, remaining: 5}]);
  assert.equal(h.elements.continue, undefined);
  assert.equal(h.navigation.length, 0);
  console.log('PASS nearly expired grant does not redirect');

  h = await harness([new Error('Simulated network failure')]);
  assert.equal(h.elements.continue, undefined);
  assert.equal(h.elements.countdown.hidden, false);
  assert.match(h.elements.countdown.textContent, /Unable to confirm/);
  assert.equal(h.navigation.length, 0);
  console.log('PASS status transport failure removes continuation and shows retry guidance');
})().catch(error => { console.error(error); process.exitCode = 1; });
