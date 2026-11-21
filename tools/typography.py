import re, glob, os
root = '.'
sources = []
html = open('index.html', encoding='utf-8').read()
sources += re.findall(r'<style[^>]*>(.*?)</style>', html, re.S)
for f in glob.glob('assets/*.css'):
    if f.endswith('typography.css'):
        continue
    sources.append(open(f, encoding='utf-8').read())
for f in glob.glob('assets/*.js') + glob.glob('teams/**/*.js', recursive=True):
    t = open(f, encoding='utf-8').read()
    # CSS kept in template literals or strings assigned to textContent / css variables
    for m in re.findall(r'`([^`]*\{[^`]*font-size:[^`]*)`', t, re.S):
        sources.append(m)
    for m in re.findall(r"'((?:[^'\\]|\\.)*\{[^']*font-size:[^']*)'", t):
        if '<svg' in m or 'class="' in m:
            continue
        sources.append(m)

def top_rules(css):
    """Yield (selector, body) for rules at depth 0, skipping @media/@keyframes/@supports blocks."""
    i, n = 0, len(css)
    while i < n:
        j = css.find('{', i)
        if j < 0:
            break
        sel = css[i:j].strip()
        depth, k = 1, j + 1
        while k < n and depth:
            if css[k] == '{': depth += 1
            elif css[k] == '}': depth -= 1
            k += 1
        body = css[j+1:k-1]
        if not sel.startswith('@'):
            yield sel, body
        i = k

def scale(px):
    if px <= 9.5: return 11.5
    if px <= 10.5: return 12
    if px <= 11.5: return 12.5
    if px <= 12: return 13
    if px <= 12.5: return 13.5
    if px <= 13: return 14
    if px <= 14: return 15
    return None

out = {}
for css in sources:
    css = re.sub(r'/\*.*?\*/', '', css, flags=re.S)
    for sel, body in top_rules(css):
        m = re.search(r'(?<![-\w])font-size:\s*([0-9.]+)px', body)
        if not m:
            continue
        new = scale(float(m.group(1)))
        if new is None:
            continue
        sel = ' '.join(sel.split())
        sel = sel.split('`')[-1].split(';')[-1].strip()
        if not sel or any(x in sel for x in ['svg', '.hld-svg', 'text.', '@', 'return', 'const ', '=>', '()', '"+', "'+"]) or not re.fullmatch(r"[\w\-\.#\[\]=\"':>+~*(),\s%^$|]+", sel):
            continue
        parts = [p.strip() for p in sel.split(',') if p.strip()]
        key = ','.join('html ' + p for p in parts)
        out[key] = max(new, out.get(key, 0))

lines = []
lines.append('/* MindSuite Pathfinder · typography layer (loaded last).')
lines.append('   One readable font (Inter, loaded in index.html; Segoe UI / system font as fallback), a 15 px base,')
lines.append('   and every small size raised: rules below are generated from the app\'s own CSS by scaling')
lines.append('   7–9.5 px → 11.5, 10–10.5 → 12, 11–11.5 → 12.5, 12 → 13, 12.5 → 13.5, 13 → 14, 14 → 15 px.')
lines.append('   Regenerate with: python tools/typography.py (run in the repo root) after adding new small text. */')
lines.append('html body{font-family:Inter,"Segoe UI Variable Text","Segoe UI",system-ui,-apple-system,Arial,sans-serif;font-size:15px;line-height:1.6;color:var(--ink);-webkit-font-smoothing:antialiased;-moz-osx-font-smoothing:grayscale;text-rendering:optimizeLegibility;font-feature-settings:"cv11","ss01"}')
lines.append('html button,html input,html select,html textarea{font-family:inherit}')
lines.append('html input,html select,html textarea{font-size:15px}')
lines.append('html small{font-size:max(12.5px,.86em)}')
lines.append('html h1{letter-spacing:-.02em;line-height:1.15}html h2{letter-spacing:-.01em;line-height:1.3}html h3{line-height:1.35}')
lines.append('html p,html li,html dd{line-height:1.6}')
lines.append('html table{font-size:14px}html th,html td{line-height:1.5}')
lines.append(':root{--muted:#5d6377}')
for k in sorted(out):
    lines.append(k + '{font-size:' + ('%g' % out[k]) + 'px}')
os.makedirs('assets', exist_ok=True)
open('assets/typography.css', 'w', encoding='utf-8', newline='').write('\n'.join(lines) + '\n')
print('rules', len(out))
