(() => {
  if ('serviceWorker' in navigator) navigator.serviceWorker.register('/app/sw.js', {scope: '/app/'}).then(registration => registration.update()).catch(() => {});
  const byId = id => document.getElementById(id);
  const generate = byId('generate'), copy = byId('copy'), result = byId('result'), message = byId('message');
  let copied = '', clearTimer = 0;
  document.addEventListener('gopass-ready', () => { byId('runtime-status').textContent = 'Ready · runs locally'; generate.disabled = false; });
  const go = new Go();
  WebAssembly.instantiateStreaming(fetch('gopass.wasm'), go.importObject).then(instance => go.run(instance.instance)).catch(error => { byId('runtime-status').textContent = 'Generator failed to load'; message.textContent = error; });
  byId('reveal').addEventListener('click', () => { const master = byId('master'); master.type = master.type === 'password' ? 'text' : 'password'; byId('reveal').textContent = master.type === 'password' ? 'Reveal' : 'Hide'; });
  generate.addEventListener('click', () => {
    const response = window.gopassGenerate(byId('site').value, byId('login').value, byId('master').value, Number(byId('length').value), Number(byId('counter').value), byId('lower').checked, byId('upper').checked, byId('digits').checked, byId('symbols').checked, byId('exclude').value);
    if (response.error) { result.textContent = 'Unable to generate'; message.textContent = response.error; copy.disabled = true; return; }
    result.textContent = response.password; message.textContent = 'Generated locally in this tab'; copy.disabled = false;
  });
  copy.addEventListener('click', async () => { await navigator.clipboard.writeText(result.textContent); copied = result.textContent; message.textContent = 'Copied; clipboard clears in 20 seconds'; clearTimeout(clearTimer); clearTimer = setTimeout(async () => { try { const current = await navigator.clipboard.readText(); if (current === copied) await navigator.clipboard.writeText(''); } catch (_) {} copied = ''; }, 20000); });
  byId('clear').addEventListener('click', () => { ['site','login','master','exclude'].forEach(id => byId(id).value = ''); result.textContent = 'Your generated password appears here'; message.textContent = 'Cleared'; copy.disabled = true; byId('master').focus(); });
})();
