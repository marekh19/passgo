(() => {
  const root = document.querySelector('[data-live][data-code]');
  if (!root || !window.EventSource) return;

  const mode = root.dataset.live;
  const code = root.dataset.code;
  const warning = document.getElementById('live-warning');
  let generation = 0;

  const showWarning = (show) => {
    if (warning) warning.hidden = !show;
  };

  const canonical = () => `/sessions/${encodeURIComponent(code)}`;

  const replaceHTML = (html) => {
    const doc = new DOMParser().parseFromString(html, 'text/html');
    if (mode === 'lobby') {
      const fresh = doc.querySelector('#roster');
      const current = document.querySelector('#roster');
      if (fresh && current) current.replaceWith(fresh);
      return;
    }
    for (const id of ['balance', 'players']) {
      const fresh = doc.querySelector(`#${id}`);
      const current = document.querySelector(`#${id}`);
      if (fresh && current) current.replaceWith(fresh);
    }
  };

  const refresh = async () => {
    const mine = ++generation;
    const path = mode === 'lobby' ? 'lobby/roster' : 'balances';
    let response;
    try {
      response = await fetch(`/sessions/${encodeURIComponent(code)}/${path}`, {
        credentials: 'same-origin',
        cache: 'no-store',
        headers: { 'Accept': 'text/html' }
      });
    } catch (_) {
      showWarning(true);
      return;
    }
    if (mine !== generation) return;
    if (response.status === 409) {
      window.location.assign(canonical());
      return;
    }
    if (response.status === 401 || response.status === 404) {
      window.location.assign(canonical());
      return;
    }
    if (!response.ok) return;
    const html = await response.text();
    if (mine !== generation) return;
    replaceHTML(html);
  };

  const events = new EventSource(`/sessions/${encodeURIComponent(code)}/events`);
  events.onopen = () => {
    showWarning(false);
    refresh();
  };
  events.onerror = () => showWarning(true);
  events.addEventListener('changed', refresh);
  events.addEventListener('started', () => window.location.assign(canonical()));
})();
