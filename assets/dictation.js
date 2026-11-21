/* Shared with the self-contained verification form. Keep this function independent of app state. */
function installTextboxDictation() {
  const selector = 'textarea,input:not([type]),input[type="text"],input[type="search"],input[type="url"],input[type="tel"],input[type="email"]';
  const Recognition = window.SpeechRecognition || window.webkitSpeechRecognition;
  let current = null;

  const style = document.createElement('style');
  style.textContent = [
    '.dictation-control{display:grid;grid-template-columns:minmax(0,1fr) auto;align-items:end;gap:4px;min-width:0;max-width:100%}',
    '.dictation-control>label{min-width:0;margin-bottom:0}.dictation-control input,.dictation-control textarea{min-width:0;max-width:100%}',
    '.dictation-control>input,.dictation-control>textarea{width:100%}.dictation-control .context-control{margin-bottom:0}',
    '.people-add>.dictation-control,.agent-conn-row>.dictation-control,.report-reader-actions>.dictation-control,.source-tools>.dictation-control,.rules-filter>.dictation-control{flex:1 1 160px}',
    '.dictation-mic{box-sizing:border-box;display:inline-grid;place-items:center;width:34px;height:34px;min-height:0;padding:0;margin:0 0 2px;border:1px solid #d7d5e5;border-radius:8px;background:#f8f7fc;color:#5546a5;cursor:pointer;line-height:1}',
    '.dictation-mic:hover,.dictation-mic:focus-visible{border-color:#6253b9}.dictation-mic:focus-visible{outline:2px solid #6253b9;outline-offset:2px}',
    '.dictation-mic[aria-pressed="true"]{background:#63365d;color:white;border-color:#63365d}.dictation-mic svg{width:17px;height:17px}',
    '.dictation-message{grid-column:1/-1;font:12px/1.45 system-ui,sans-serif;color:#51465f;overflow-wrap:anywhere}.dictation-message:empty,.dictation-mic[hidden]{display:none}',
    '@media print{.dictation-control{display:block}.dictation-mic,.dictation-message{display:none!important}}'
  ].join('\n');
  document.head.append(style);

  function editable(field) {
    return field.matches(selector) && !field.matches(':disabled,[readonly],[hidden]') &&
      !/(?:password|one-time-code|cc-number|cc-csc)/i.test(field.autocomplete || '');
  }
  function available(field) {
    return field.isConnected && editable(field) && field.getClientRects().length &&
      !field.closest('[inert],dialog:not([open])');
  }
  function message(control, text, clear = false) {
    clearTimeout(control.timer);
    control.status.textContent = text;
    if (clear) control.timer = setTimeout(() => { control.status.textContent = ''; }, 7000);
  }
  function finish(session, text, abort = false) {
    if (current !== session) return;
    current = null;
    session.control.button.setAttribute('aria-pressed', 'false');
    session.control.button.title = session.control.idleTitle;
    message(session.control, text, true);
    session.control.field.dispatchEvent(new Event('dictationend'));
    if (abort) {
      try { session.engine.abort(); } catch { /* The recognizer may have already ended. */ }
    }
  }
  function fallback() {
    if (/Windows/i.test(navigator.userAgent)) return 'To dictate here, press Windows key + H.';
    if (/Macintosh/i.test(navigator.userAgent) && navigator.maxTouchPoints < 2) return 'To dictate here, use the Dictation shortcut in your Mac keyboard settings.';
    return 'To dictate here, use the microphone on your device’s keyboard or its voice typing shortcut.';
  }
  function insert(field, words) {
    const value = field.value;
    const start = field.selectionStart ?? value.length;
    const end = field.selectionEnd ?? start;
    const left = value.slice(0, start), right = value.slice(end);
    const prose = ['text','search','textarea'].includes(field.type);
    const before = prose && left && !/[\s([{“‘'-]$/u.test(left) && !/^[\s.,!?;:)\]}’']/u.test(words) ? ' ' : '';
    const after = prose && right && !/^[\s.,!?;:)\]}’']/u.test(right) && !/[\s([{“‘'-]$/u.test(words) ? ' ' : '';
    let addition = before + words + after;
    if (field.maxLength >= 0) {
      const room = Math.max(0, field.maxLength - left.length - right.length);
      addition = addition.slice(0, room).replace(/[\uD800-\uDBFF]$/, '');
    }
    if (!addition.trim()) return false;
    if (field.selectionStart == null) field.value = left + addition + right;
    else {
      field.setRangeText(addition, start, end, 'end');
      if (after && addition.endsWith(after)) field.setSelectionRange(start + addition.length - 1, start + addition.length - 1);
    }
    field.dispatchEvent(new Event('input', {bubbles:true}));
    field.dispatchEvent(new Event('change', {bubbles:true}));
    return true;
  }
  function toggle(control) {
    const {field, button} = control;
    if (!available(field)) return;
    if (current?.control === control) {
      const session = current;
      if (!session.stopping) {
        session.stopping = true;
        message(control, 'Finishing dictation…');
        try { session.engine.stop(); } catch { finish(session, 'Dictation stopped.', true); }
      }
      return;
    }
    if (current) finish(current, '', true);
    field.focus({preventScroll:true});
    if (!Recognition) { message(control, fallback(), true); return; }
    let engine;
    try { engine = new Recognition(); }
    catch { message(control, fallback(), true); return; }
    const session = {control, engine, nextFinal:0, stopping:false};
    current = session;
    engine.lang = navigator.language || document.documentElement.lang || 'en';
    engine.continuous = true;
    engine.interimResults = true;
    button.setAttribute('aria-pressed', 'true');
    button.title = 'Stop dictation';
    message(control, 'Starting microphone…');
    engine.onstart = () => {
      if (current === session && !session.stopping) message(control, 'Listening. Select the microphone again to stop.');
    };
    engine.onresult = event => {
      if (current !== session) return;
      if (!available(field)) { finish(session, '', true); return; }
      const words = [], preview = [];
      for (let i = session.nextFinal; i < event.results.length; i++) {
        const result = event.results[i];
        const transcript = result[0].transcript.trim();
        if (result.isFinal) {
          session.nextFinal = i + 1;
          if (transcript) words.push(transcript);
        } else if (transcript) preview.push(transcript);
      }
      if (words.length && !insert(field, words.join(' '))) {
        finish(session, 'This field has reached its character limit.', true);
        return;
      }
      if (!available(field)) { finish(session, '', true); return; }
      if (!session.stopping) message(control, preview.join(' ') || 'Listening. Select the microphone again to stop.');
    };
    engine.onerror = event => {
      const errors = {
        'not-allowed':'Allow microphone access in your browser’s site settings to use dictation.',
        'audio-capture':'Check that your microphone is connected and available.',
        'no-speech':'No speech heard. Select the microphone to try again.',
        'aborted':'Dictation stopped.'
      };
      finish(session, errors[event.error] || 'Speech recognition is unavailable. ' + fallback());
    };
    engine.onend = () => finish(session, 'Dictation stopped.');
    try { engine.start(); } catch { finish(session, fallback(), true); }
  }

  function decorate(field) {
    if (field.closest('.dictation-control') || !editable(field)) return;
    const label = field.closest('label');
    const host = label && label.querySelectorAll('input,textarea,select').length === 1 ? label : field;
    const wrapper = document.createElement('span');
    wrapper.className = 'dictation-control';
    const name = field.getAttribute('aria-label') || field.labels?.[0]?.innerText?.trim() || field.placeholder || 'this field';
    const button = document.createElement('button');
    button.type = 'button';
    button.className = 'dictation-mic';
    button.setAttribute('aria-label', 'Dictate into ' + name);
    button.setAttribute('aria-pressed', 'false');
    button.title = 'Dictate into ' + name;
    const ns = 'http://www.w3.org/2000/svg';
    const svg = document.createElementNS(ns, 'svg');
    for (const [k,v] of Object.entries({viewBox:'0 0 24 24',fill:'none',stroke:'currentColor','stroke-width':'1.8','stroke-linecap':'round','aria-hidden':'true'})) svg.setAttribute(k,v);
    const path = document.createElementNS(ns, 'path');
    path.setAttribute('d','M9 5a3 3 0 0 1 6 0v7a3 3 0 0 1-6 0zM5.5 11v1a6.5 6.5 0 0 0 13 0v-1M12 19v3m-3 0h6');
    svg.append(path);
    button.append(svg);
    const status = document.createElement('span');
    status.className = 'dictation-message';
    status.setAttribute('role','status');
    const control = {field, button, status, idleTitle:button.title};
    button.addEventListener('mousedown', event => { if (document.activeElement === field) event.preventDefault(); });
    button.addEventListener('click', event => { event.preventDefault(); toggle(control); });
    const focused = document.activeElement === field;
    host.before(wrapper);
    wrapper.append(host, button, status);
    if (focused) field.focus({preventScroll:true});
  }

  function scan(node) {
    if (!(node instanceof Element)) return;
    if (node.matches(selector)) decorate(node);
    else node.querySelectorAll(selector).forEach(decorate);
  }
  scan(document.body);
  new MutationObserver(records => {
    for (const record of records) {
      if (record.type === 'childList') record.addedNodes.forEach(scan);
      else if (record.target.matches('input,textarea')) {
        const field = record.target;
        const wrapper = field.closest('.dictation-control');
        if (wrapper) wrapper.querySelector('.dictation-mic').hidden = !editable(field);
        else if (editable(field)) decorate(field);
      }
    }
    if (current && !available(current.control.field)) finish(current, '', true);
  }).observe(document.body, {childList:true, subtree:true, attributes:true, attributeFilter:['open','hidden','disabled','readonly','type','inert']});
  document.addEventListener('focusin', event => {
    if (current && event.target !== current.control.field && event.target !== current.control.button) finish(current, '', true);
  });
  document.addEventListener('keydown', event => {
    if (current && event.key === 'Escape') {
      event.preventDefault();
      event.stopPropagation();
      finish(current, 'Dictation stopped.', true);
    }
  }, true);
  document.addEventListener('visibilitychange', () => { if (document.hidden && current) finish(current, '', true); });
  window.addEventListener('pagehide', () => { if (current) finish(current, '', true); });
}
installTextboxDictation();

