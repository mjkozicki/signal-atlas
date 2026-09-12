// The public guide is an illustration and glossary; it never calls scanner APIs.
export function relativePower(signal) {
  return 10 ** ((signal + 60) / 10);
}

export function matchesTerm(term, query, category) {
  return (category === 'All' || term.category === category)
    && term.text.toLowerCase().includes(query.trim().toLowerCase());
}

if (typeof document !== 'undefined') {
  const signal = document.getElementById('example-rssi');
  if (signal) {
    const output = document.getElementById('signal-value');
    const power = document.getElementById('relative-power');
    const format = new Intl.NumberFormat(undefined, { maximumSignificantDigits: 2 });
    const update = () => {
      const value = Number(signal.value);
      output.textContent = `${value} dBm`;
      signal.setAttribute('aria-valuetext', `${value} dBm`);
      power.textContent = `${format.format(relativePower(value))}×`;
    };
    signal.disabled = false;
    signal.addEventListener('input', update);
    update();
  }

  const query = document.getElementById('glossary-query');
  if (query) {
    const terms = [...document.querySelectorAll('.term')].map(element => ({
      element, category: element.dataset.category, text: element.textContent,
    }));
    const buttons = [...document.querySelectorAll('[data-category-filter]')];
    const count = document.getElementById('glossary-count');
    const empty = document.getElementById('glossary-empty');
    let category = 'All';
    const update = () => {
      let visible = 0;
      for (const term of terms) {
        term.element.hidden = !matchesTerm(term, query.value, category);
        if (!term.element.hidden) visible++;
      }
      for (const button of buttons) {
        button.setAttribute('aria-pressed', String(button.dataset.categoryFilter === category));
      }
      count.textContent = `${visible} ${visible === 1 ? 'term' : 'terms'}`;
      empty.hidden = visible !== 0;
    };
    const clear = () => {
      query.value = '';
      category = 'All';
      update();
    };
    query.addEventListener('input', update);
    for (const button of buttons) {
      button.addEventListener('click', () => {
        category = button.dataset.categoryFilter;
        update();
      });
    }
    document.getElementById('clear-filters').addEventListener('click', () => {
      clear();
      query.focus();
    });
    const openLinkedTerm = () => {
      let id;
      try { id = decodeURIComponent(window.location.hash.slice(1)); } catch { return; }
      const term = terms.find(item => item.element.id === id);
      if (term) {
        clear();
        term.element.open = true;
        term.element.scrollIntoView({ block: 'start' });
      }
    };
    document.getElementById('glossary-tools').hidden = false;
    window.addEventListener('hashchange', openLinkedTerm);
    update();
    openLinkedTerm();
  }
}
