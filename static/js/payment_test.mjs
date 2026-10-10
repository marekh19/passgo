import { createRequire } from 'node:module';
import test from 'node:test';
import assert from 'node:assert/strict';
const { amountState, editAmount, init } = createRequire(import.meta.url)('./payment.js');

test('amount validity, exact range, normalization and formatting', () => {
  for (const raw of ['', '0', '000', '-1', '1.5', ' 50', '１２', '1e3', '18446744073709551616', '9'.repeat(10000)]) {
    assert.equal(amountState(raw).valid, false, raw.slice(0, 30));
  }
  assert.equal(amountState('000200').normalized, '200');
  assert.equal(amountState('1500').formatted, '$1,500');
  assert.equal(amountState('18446744073709551615').formatted, '$18,446,744,073,709,551,615');
  assert.equal(amountState('18446744073709551615').valid, true);
  for (const value of ['50', '100', '200', '500']) assert.equal(amountState(value).valid, true);
});

test('keypad inserts and deletes at selection without changing other content', () => {
  assert.deepEqual(editAmount('123', 1, 2, '9'), { value: '193', caret: 2 });
  assert.deepEqual(editAmount('123', 1, 1, '9'), { value: '1923', caret: 2 });
  assert.deepEqual(editAmount('123', 1, 3, null), { value: '1', caret: 1 });
  assert.deepEqual(editAmount('123', 2, 2, null), { value: '13', caret: 1 });
  assert.deepEqual(editAmount('123', 0, 0, null), { value: '123', caret: 0 });
});

class Element {
  constructor(dataset = {}) {
    this.dataset = dataset;
    this.listeners = {};
    this.attrs = {};
    this.value = '';
    this.selectionStart = this.selectionEnd = 0;
    const classes = new Set();
    this.classList = {
      toggle(name, enabled) { if (enabled) classes.add(name); else classes.delete(name); },
      add(name) { classes.add(name); },
      remove(name) { classes.delete(name); },
      contains(name) { return classes.has(name); },
    };
    const properties = new Map();
    this.style = {
      setProperty(name, value) { properties.set(name, value); },
      getPropertyValue(name) { return properties.get(name) || ''; },
      removeProperty(name) { properties.delete(name); },
    };
  }
  addEventListener(name, fn) { (this.listeners[name] ??= []).push(fn); }
  fire(name) {
    const event = { prevented: false, preventDefault() { this.prevented = true; } };
    (this.listeners[name] || []).forEach(fn => fn(event));
    return event;
  }
  setAttribute(name, value) { this.attrs[name] = value; }
  setSelectionRange(start, end) { this.selectionStart = start; this.selectionEnd = end; }
  focus(options) { this.document.activeElement = this; this.focusOptions = options; }
}
function fixture() {
  const document = {
    documentElement: new Element(),
    defaultView: {
      scrollX: 0, scrollY: 450,
      scrollTo(x, y) { this.scrollX = x; this.scrollY = y; },
    },
  };
  const field = new Element();
  const form = new Element();
  const submit = new Element({ action: 'Pay' });
  const error = new Element();
  const display = new Element();
  const result = new Element();
  const close = new Element();
  const trigger = new Element({ sheetOpen: 'demo' });
  const presets = ['50', '100', '200', '500'].map(preset => new Element({ preset }));
  const keys = [new Element({ digit: '9' }), new Element({ backspace: '' })];
  const sheet = new Element();
  sheet.id = 'demo';
  const selectors = {
    '[data-payment-form]': form, '[data-amount-field]': field,
    '[data-amount-display]': display, '[data-amount-error]': error,
    '[data-payment-submit]': submit, '[data-demo-result]': result,
    '[data-sheet-close]': close,
  };
  sheet.querySelector = selector => selectors[selector];
  sheet.querySelectorAll = selector => selector === '[data-preset]' ? presets : keys;
  sheet.showModal = () => { sheet.open = true; };
  sheet.close = () => { sheet.open = false; sheet.fire('close'); };
  document.querySelectorAll = selector => selector === '[data-payment-demo]' ? [sheet] : [trigger];
  [field, trigger].forEach(element => { element.document = document; });
  init(document);
  return { document, sheet, field, form, submit, error, display, result, close, trigger, presets, keys };
}

test('open, editable input, preset, invalid paste, pending submit, close and reopen', () => {
  const f = fixture();
  f.trigger.fire('click');
  assert.equal(f.sheet.open, true);
  assert.equal(f.document.activeElement, f.field);
  assert.equal(f.submit.disabled, true);
  f.presets[2].fire('click');
  assert.equal(f.field.value, '200');
  assert.equal(f.presets[2].attrs['aria-pressed'], 'true');
  assert.equal(f.submit.textContent, 'Pay $200');
  f.field.value = '2.50';
  f.field.fire('input');
  assert.equal(f.field.value, '2.50');
  assert.equal(f.error.hidden, false);
  assert.equal(f.submit.disabled, true);
  assert.equal(f.form.fire('submit').prevented, true);
  f.field.value = '0001500';
  f.field.fire('input');
  f.field.fire('blur');
  assert.equal(f.field.value, '1500');
  assert.equal(f.form.fire('submit').prevented, true);
  assert.equal(f.submit.textContent, 'Paying $1,500…');
  assert.equal(f.submit.attrs['aria-busy'], 'true');
  assert.equal(f.field.readOnly, true);
  const message = f.result.textContent;
  f.form.fire('submit');
  f.presets[0].fire('click');
  assert.equal(f.field.value, '1500');
  assert.equal(f.result.textContent, message);
  f.close.fire('click');
  assert.equal(f.document.activeElement, f.trigger);
  f.trigger.fire('click');
  assert.equal(f.field.value, '');
  assert.equal(f.field.readOnly, false);
  assert.equal(f.result.textContent, '');
});

test('pointer keypad preserves selection, keyboard keypad remains focusable, init is idempotent', () => {
  const f = fixture();
  init(f.document);
  assert.equal(f.trigger.listeners.click.length, 1);
  f.trigger.fire('click');
  f.field.value = '123';
  f.field.setSelectionRange(1, 2);
  assert.equal(f.keys[0].fire('pointerdown').prevented, true);
  f.keys[0].fire('click');
  assert.equal(f.field.value, '193');
  f.keys[1].fire('click');
  assert.equal(f.field.value, '13');
  f.document.activeElement = f.keys[0];
  assert.equal(f.keys[0].fire('pointerdown').prevented, false);
});


test('sheet locks the background and restores page position on close and reopen', () => {
  const f = fixture();
  const root = f.document.documentElement;
  f.trigger.fire('click');
  assert.equal(root.classList.contains('pg-scroll-locked'), true);
  assert.equal(root.style.getPropertyValue('--pg-scroll-top'), '-450px');
  f.document.defaultView.scrollY = 0;
  f.close.fire('click');
  assert.equal(root.classList.contains('pg-scroll-locked'), false);
  assert.equal(root.style.getPropertyValue('--pg-scroll-top'), '');
  assert.equal(f.document.defaultView.scrollY, 450);
  assert.deepEqual(f.trigger.focusOptions, { preventScroll: true });
  f.document.defaultView.scrollY = 700;
  f.trigger.fire('click');
  assert.equal(root.style.getPropertyValue('--pg-scroll-top'), '-700px');
  f.sheet.close();
  assert.equal(f.document.defaultView.scrollY, 700);
});

test('failed dialog opening does not leave the background locked', () => {
  const f = fixture();
  f.sheet.showModal = () => { throw new Error('cannot open dialog'); };
  assert.throws(() => f.trigger.fire('click'), /cannot open dialog/);
  assert.equal(f.document.documentElement.classList.contains('pg-scroll-locked'), false);
  assert.equal(f.document.documentElement.style.getPropertyValue('--pg-scroll-top'), '');
  assert.equal(f.document.defaultView.scrollY, 450);
});
