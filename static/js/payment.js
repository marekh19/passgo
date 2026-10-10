(function (root) {
  "use strict";
  const MAX = 18446744073709551615n;

  function amountState(raw) {
    if (raw === "") return { valid: false, normalized: "", formatted: "$0", error: "" };
    if (!/^[0-9]+$/.test(raw)) return { valid: false, error: "Enter whole dollars using digits 0–9 only." };
    const normalized = raw.replace(/^0+(?=\d)/, "");
    // Compare lengths first so pasted input cannot cause an unbounded BigInt conversion.
    if (normalized.length > 20 || BigInt(normalized) > MAX) {
      return { valid: false, error: "Enter no more than $18,446,744,073,709,551,615." };
    }
    return {
      valid: normalized !== "0", normalized,
      formatted: "$" + normalized.replace(/\B(?=(\d{3})+(?!\d))/g, ","),
      error: normalized === "0" ? "Enter an amount greater than zero." : "",
    };
  }

  function editAmount(value, start, end, digit) {
    if (digit === null) {
      const from = start === end ? Math.max(0, start - 1) : start;
      return { value: value.slice(0, from) + value.slice(end), caret: from };
    }
    return { value: value.slice(0, start) + digit + value.slice(end), caret: start + 1 };
  }

  function initSheet(sheet, document) {
    if (sheet.dataset.paymentReady) return;
    sheet.dataset.paymentReady = "true";
    const form = sheet.querySelector("[data-payment-form]");
    const field = sheet.querySelector("[data-amount-field]");
    const display = sheet.querySelector("[data-amount-display]");
    const error = sheet.querySelector("[data-amount-error]");
    const submit = sheet.querySelector("[data-payment-submit]");
    const result = sheet.querySelector("[data-demo-result]");
    const presets = [...sheet.querySelectorAll("[data-preset]")];
    const keys = [...sheet.querySelectorAll("[data-digit], [data-backspace]")];
    let selection = [0, 0];
    let trigger;
    let pending = false;
    let scrollPosition;
    const page = document.documentElement;
    const view = document.defaultView;
    const action = submit.dataset.action || "Pay";

    function lockBackground() {
      scrollPosition = [view.scrollX, view.scrollY];
      page.style.setProperty("--pg-scroll-top", `${-view.scrollY}px`);
      page.classList.add("pg-scroll-locked");
    }
    function unlockBackground() {
      if (!scrollPosition) return;
      page.classList.remove("pg-scroll-locked");
      page.style.removeProperty("--pg-scroll-top");
      view.scrollTo(...scrollPosition);
      scrollPosition = undefined;
    }
    function saveSelection() {
      selection = [field.selectionStart ?? field.value.length, field.selectionEnd ?? field.value.length];
    }
    function update() {
      const state = amountState(field.value);
      error.textContent = state.error;
      error.hidden = !state.error;
      field.setAttribute("aria-invalid", String(Boolean(state.error)));
      display.textContent = state.formatted || "Invalid amount";
      display.classList.toggle("pg-money-long", (state.formatted || "").length > 14);
      submit.disabled = pending || !state.valid;
      submit.setAttribute("aria-busy", String(pending));
      submit.textContent = pending ? `${action === "Pay" ? "Paying" : "Collecting"} ${state.formatted}…` : `${action} ${state.formatted || "amount"}`;
      presets.forEach((button) => button.setAttribute("aria-pressed", String(state.valid && state.normalized === button.dataset.preset)));
      [...presets, ...keys].forEach((button) => { button.disabled = pending; });
      field.readOnly = pending;
      return state;
    }
    function normalize() {
      const state = amountState(field.value);
      if (state.normalized !== undefined && field.value !== "") field.value = state.normalized;
      update();
      saveSelection();
    }
    function change(value, caret) {
      field.value = value;
      field.setSelectionRange(caret, caret);
      selection = [caret, caret];
      result.textContent = "";
      update();
    }
    field.addEventListener("input", () => { saveSelection(); result.textContent = ""; update(); });
    ["select", "keyup", "click"].forEach((event) => field.addEventListener(event, saveSelection));
    field.addEventListener("blur", normalize);
    [...presets, ...keys].forEach((button) => {
      button.addEventListener("pointerdown", (event) => {
        if (document.activeElement === field) {
          saveSelection();
          event.preventDefault();
        }
      });
      button.addEventListener("click", () => {
        if (pending) return;
        if (button.dataset.preset) {
          change(button.dataset.preset, button.dataset.preset.length);
          return;
        }
        const edited = editAmount(field.value, ...selection, button.dataset.digit ?? null);
        change(edited.value, edited.caret);
      });
    });
    sheet.querySelector("[data-sheet-close]").addEventListener("click", () => sheet.close());
    sheet.addEventListener("close", () => {
      unlockBackground();
      if (trigger) trigger.focus({ preventScroll: true });
    });
    form.addEventListener("submit", (event) => {
      // Preview-only forms never submit, including invalid or repeated activation.
      event.preventDefault();
      if (pending || !amountState(field.value).valid) return;
      normalize();
      pending = true;
      update();
      result.textContent = "Demo only. No money moved. Close and reopen to try again.";
    });
    const triggers = [...document.querySelectorAll("[data-sheet-open]")].filter((button) => button.dataset.sheetOpen === sheet.id);
    triggers.forEach((button) => button.addEventListener("click", () => {
      if (sheet.open) return;
      trigger = button;
      pending = false;
      field.value = "";
      selection = [0, 0];
      result.textContent = "";
      update();
      lockBackground();
      try {
        sheet.showModal();
        field.focus();
      } catch (error) {
        unlockBackground();
        throw error;
      }
    }));
    update();
  }

  function init(document) {
    document.querySelectorAll("[data-payment-demo]").forEach((sheet) => initSheet(sheet, document));
  }
  if (typeof module !== "undefined") module.exports = { amountState, editAmount, init };
  if (root.document) init(root.document);
})(typeof window === "undefined" ? globalThis : window);
