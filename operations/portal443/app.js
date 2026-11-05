'use strict';
// Every countdown step rechecks server status; a stale tab cannot extend access.
let countdown = 5;
let started = false;
async function tick() {
  try {
    const response = await fetch('/portal/status', {cache: 'no-store', credentials: 'same-origin'});
    if (!response.ok) throw new Error('Session unavailable');
    const status = await response.json();
    if (status.state === 'pending') {
      document.getElementById('state').textContent = 'Authorizing your IP...';
      setTimeout(tick, 1000);
      return;
    }
    if (status.state !== 'active' || status.remaining <= 5) {
      document.getElementById('state').textContent = status.message || 'Access is no longer active.';
      document.getElementById('countdown').textContent = '';
      document.getElementById('continue')?.remove();
      return;
    }
    if (!document.getElementById('continue')) {
      // Reload for server-rendered success details and safe no-JavaScript link.
      location.replace('/portal/');
      return;
    }
    document.getElementById('state').textContent = 'Access granted';
    const minutes = Math.floor(status.remaining / 60);
    document.getElementById('lifetime').textContent = `${Math.floor(minutes / 60)}h ${minutes % 60}m`;
    if (started) countdown--;
    started = true;
    document.getElementById('countdown').textContent = `Redirecting in: ${countdown} seconds`;
    document.getElementById('countdown').hidden = false;
    if (countdown <= 0) {
      location.assign('/portal/continue'); // Server rechecks; destination is never supplied by the browser.
      return;
    }
    setTimeout(tick, 1000);
  } catch (_) {
    document.getElementById('countdown').textContent = 'Unable to confirm access. Refresh to retry.';
    document.getElementById('countdown').hidden = false;
    document.getElementById('continue')?.remove();
  }
}
tick();
