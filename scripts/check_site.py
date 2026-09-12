#!/usr/bin/env python3
"""Check public page links, accessible references, and glossary parity with the app."""
from html.parser import HTMLParser
from pathlib import Path
import re
from urllib.parse import unquote, urlsplit

ROOT = Path(__file__).resolve().parents[1]
SITE = ROOT / 'src/site'


class Page(HTMLParser):
    def __init__(self, path):
        super().__init__()
        self.path = path
        self.ids = set()
        self.refs = []
        self.labels = []
        self.h1s = 0
        self.canonical = None
        self.current_links = 0
        self.terms = []
        self.term = None
        self.field = None

    def handle_starttag(self, tag, attrs):
        attrs = dict(attrs)
        if 'id' in attrs:
            assert attrs['id'] not in self.ids, (self.path, 'duplicate ID', attrs['id'])
            self.ids.add(attrs['id'])
        self.refs.extend(attrs[key] for key in ('href', 'src') if key in attrs)
        for key in ('aria-labelledby', 'aria-describedby'):
            self.labels.extend(attrs.get(key, '').split())
        if tag == 'label' and 'for' in attrs:
            self.labels.append(attrs['for'])
        if tag == 'h1':
            self.h1s += 1
        if tag == 'link' and attrs.get('rel') == 'canonical':
            self.canonical = attrs['href']
        if attrs.get('aria-current') == 'page':
            self.current_links += 1
        if tag == 'img':
            assert {'alt', 'width', 'height'} <= attrs.keys(), (self.path, 'image attributes')
        if tag == 'details' and 'data-category' in attrs:
            self.term = {'category': attrs['data-category'], 'name': '', 'text': ''}
        if self.term is not None and tag in ('span', 'p'):
            self.field = 'name' if tag == 'span' else 'text'

    def handle_endtag(self, tag):
        if self.term is not None:
            if tag in ('span', 'p'):
                self.field = None
            if tag == 'details':
                self.terms.append(self.term)
                self.term = None
                self.field = None

    def handle_data(self, data):
        if self.term is not None and self.field:
            self.term[self.field] += data


def main():
    pages = {}
    for path in SITE.glob('*.html'):
        source = path.read_text()
        assert not re.search(r'bind:|\{#|\{/|<Radio\b|<LockKeyhole\b', source), path
        page = Page(path)
        page.feed(source)
        assert page.h1s == 1, (path, 'one main heading required')
        assert page.current_links >= 1, (path, 'active navigation missing')
        suffix = '' if path.name == 'index.html' else path.name
        assert page.canonical == 'https://mjkozicki.github.io/signal-atlas/' + suffix, path
        assert set(page.labels) <= page.ids, (path, 'missing accessible label target')
        pages[path.resolve()] = page

    def check_ref(path, ref):
        url = urlsplit(ref)
        if url.scheme or url.netloc:
            return
        target = (path.parent / unquote(url.path)).resolve() if url.path else path.resolve()
        if target.is_dir():
            target /= 'index.html'
        assert target.is_relative_to(SITE.resolve()), (path, 'asset outside published directory', ref)
        assert target.exists(), (path, 'missing target', ref)
        if url.fragment and target in pages:
            assert unquote(url.fragment) in pages[target].ids, (path, 'missing anchor', ref)

    for path, page in pages.items():
        for ref in page.refs:
            check_ref(path, ref)
    # Check destinations used by old homepage bookmarks, too.
    for target in re.findall(r"'#[^']+': '([^']+)'", (SITE / 'site.js').read_text()):
        check_ref(SITE / 'index.html', target)

    app = (ROOT / 'src/web/src/components/About.svelte').read_text()
    expected = [dict(zip(('category', 'name', 'text'), match)) for match in re.findall(
        r"\{ category: '([^']+)', name: '([^']+)', text: '([^']+)' \}", app)]
    actual = pages[(SITE / 'glossary.html').resolve()].terms
    assert expected and actual == expected, 'Public glossary differs from the app About guide'
    print(f'{len(pages)} pages: links, assets, headings, navigation, labels, and {len(actual)} glossary definitions pass')


if __name__ == '__main__':
    main()
