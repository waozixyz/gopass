(() => {
  const byId = id => document.getElementById(id);
  const card = byId('generator');
  const generate = byId('generate'), copy = byId('copy'), result = byId('result'), message = byId('message');
  let copied = '', clearTimer = 0, started = false, ready = false, pending = false;

  function runGenerate() {
    const response = window.passGenerate(byId('site').value, byId('login').value, byId('master').value, Number(byId('length').value), Number(byId('counter').value), byId('lower').checked, byId('upper').checked, byId('digits').checked, byId('symbols').checked, byId('exclude').value);
    if (response.error) { result.textContent = 'Unable to generate'; message.textContent = response.error; copy.disabled = true; return; }
    result.textContent = response.password; message.textContent = 'Generated locally in this tab'; copy.disabled = false;
  }

  function startLoad() {
    if (started) return;
    started = true;
    const script = document.createElement('script');
    script.src = '/app/wasm_exec.js';
    script.onload = () => {
      const go = new Go();
      WebAssembly.instantiateStreaming(fetch('/app/pass.wasm'), go.importObject).then(instance => go.run(instance.instance)).catch(error => { message.textContent = 'Generator failed to load'; generate.textContent = error; });
    };
    script.onerror = () => { message.textContent = 'Generator failed to load'; };
    document.head.appendChild(script);
  }

  ['focusin', 'input', 'click'].forEach(type => card.addEventListener(type, startLoad, {passive: true, capture: true}));
  document.addEventListener('pass-ready', () => { ready = true; generate.disabled = false; if (pending) { pending = false; runGenerate(); } });

  byId('reveal').addEventListener('click', () => { const master = byId('master'); master.type = master.type === 'password' ? 'text' : 'password'; byId('reveal').textContent = master.type === 'password' ? 'Reveal' : 'Hide'; });
  generate.addEventListener('click', () => { if (!ready) { pending = true; message.textContent = 'Loading secure generator…'; return; } runGenerate(); });
  copy.addEventListener('click', async () => { await navigator.clipboard.writeText(result.textContent); copied = result.textContent; message.textContent = 'Copied; clipboard clears in 20 seconds'; clearTimeout(clearTimer); clearTimer = setTimeout(async () => { try { const current = await navigator.clipboard.readText(); if (current === copied) await navigator.clipboard.writeText(''); } catch (_) {} copied = ''; }, 20000); });
  byId('clear').addEventListener('click', () => { ['site','login','master','exclude'].forEach(id => byId(id).value = ''); result.textContent = 'Your generated password appears here'; message.textContent = 'Cleared'; copy.disabled = true; byId('site').focus(); });
})();
