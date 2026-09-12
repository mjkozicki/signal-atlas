// Keep links to the former one-page site's sections useful after the split.
if (window.location.pathname.endsWith('/') || window.location.pathname.endsWith('/index.html')) {
  const moved = {
    '#capabilities': 'scanners.html#capabilities',
    '#workspace': 'scanners.html#workspace',
    '#wifi': 'scanners.html#wifi',
    '#bluetooth': 'scanners.html#bluetooth',
    '#nfc': 'scanners.html#nfc',
    '#quickstart': 'quickstart.html',
    '#about': 'about.html',
    '#measurements': 'measurements.html',
    '#scan-process': 'process.html',
    '#glossary': 'glossary.html',
    '#data-limits': 'about.html#data-limits',
  };
  const followMovedSection = () => {
    const destination = moved[window.location.hash];
    if (destination) window.location.replace(new URL(destination, window.location.href));
  };
  window.addEventListener('hashchange', followMovedSection);
  followMovedSection();
}

for (const button of document.querySelectorAll('[data-copy]')) {
  if (!navigator.clipboard || !window.isSecureContext) continue;
  button.hidden = false;
  button.addEventListener('click', async () => {
    const code = document.getElementById(button.dataset.copy);
    const status = document.getElementById('copy-status');
    try {
      await navigator.clipboard.writeText(code.textContent.trim());
      status.textContent = 'Commands copied. Paste them into your terminal.';
      button.textContent = 'Copied';
      window.setTimeout(() => { button.textContent = 'Copy'; }, 2000);
    } catch {
      status.textContent = 'Clipboard access is unavailable. Select and copy the commands above.';
    }
  });
}
