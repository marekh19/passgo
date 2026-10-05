import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';
import vm from 'node:vm';

const scriptSource = await readFile(new URL('./live.js', import.meta.url), 'utf8');

function deferred() {
  let resolve;
  const promise = new Promise((done) => {
    resolve = done;
  });
  return { promise, resolve };
}

async function loadLive({ mode = 'game', fetches = [] } = {}) {
  const replacements = [];
  const fetchCalls = [];
  const listeners = new Map();
  const warning = { hidden: true };
  const root = { dataset: { live: mode, code: 'ABCD' } };
  const nodes = new Map([
    ['#balance', { replaceWith: (node) => replacements.push(node.html) }],
    ['#players', { replaceWith: (node) => replacements.push(node.html) }],
    ['#roster', { replaceWith: (node) => replacements.push(node.html) }],
  ]);

  globalThis.window = {
    EventSource: class FakeEventSource {
      constructor(url) {
        this.url = url;
        globalThis.__events = this;
      }
      addEventListener(name, fn) {
        listeners.set(name, fn);
      }
    },
    location: { assigned: '', assign(url) { this.assigned = url; } },
  };
  globalThis.EventSource = globalThis.window.EventSource;
  globalThis.document = {
    querySelector(selector) {
      if (selector === '[data-live][data-code]') return root;
      return nodes.get(selector) ?? null;
    },
    getElementById(id) {
      return id === 'live-warning' ? warning : null;
    },
  };
  globalThis.DOMParser = class FakeDOMParser {
    parseFromString(html) {
      return {
        querySelector(selector) {
          if (html.includes(selector.slice(1))) return { html };
          return null;
        },
      };
    }
  };
  globalThis.fetch = async (url, options) => {
    fetchCalls.push({ url, options });
    const next = fetches.shift();
    if (!next) throw new Error('unexpected fetch');
    if (next.reject) throw next.reject;
    return next;
  };

  vm.runInThisContext(scriptSource, { filename: 'static/js/live.js' });
  return {
    warning,
    fetchCalls,
    replacements,
    open: () => globalThis.__events.onopen(),
    error: () => globalThis.__events.onerror(),
    changed: () => listeners.get('changed')(),
  };
}

test('older refresh cannot overwrite a newer snapshot', async () => {
  const oldBody = deferred();
  const freshBody = deferred();
  const live = await loadLive({
    fetches: [
      { status: 200, ok: true, text: () => oldBody.promise },
      { status: 200, ok: true, text: () => freshBody.promise },
    ],
  });

  live.changed();
  await Promise.resolve();
  live.changed();
  await Promise.resolve();

  freshBody.resolve('<section id="balance">fresh</section><section id="players">fresh</section>');
  await Promise.resolve();
  await Promise.resolve();

  oldBody.resolve('<section id="balance">old</section><section id="players">old</section>');
  await Promise.resolve();
  await Promise.resolve();

  assert.deepEqual(live.replacements, [
    '<section id="balance">fresh</section><section id="players">fresh</section>',
    '<section id="balance">fresh</section><section id="players">fresh</section>',
  ]);
});

test('open and reopen hide warning and fetch a fresh snapshot', async () => {
  const live = await loadLive({
    mode: 'lobby',
    fetches: [
      { status: 200, ok: true, text: async () => '<ul id="roster">first</ul>' },
      { status: 200, ok: true, text: async () => '<ul id="roster">second</ul>' },
    ],
  });

  live.error();
  assert.equal(live.warning.hidden, false);

  live.open();
  await Promise.resolve();
  await Promise.resolve();
  assert.equal(live.warning.hidden, true);

  live.error();
  live.open();
  await Promise.resolve();
  await Promise.resolve();

  assert.equal(live.fetchCalls.length, 2);
  assert.deepEqual(live.replacements, ['<ul id="roster">first</ul>', '<ul id="roster">second</ul>']);
});
