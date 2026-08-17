const cacheName = 'gopass-app-v1';
const appFiles = ['/app/', '/app/app.js', '/app/wasm_exec.js', '/app/gopass.wasm', '/styles.css', '/manifest.webmanifest', '/icons/icon-192.png', '/icons/icon-512.png'];
self.addEventListener('install', event => event.waitUntil(caches.open(cacheName).then(cache => cache.addAll(appFiles))));
self.addEventListener('activate', event => event.waitUntil(caches.keys().then(keys => Promise.all(keys.filter(key => key !== cacheName).map(key => caches.delete(key))))));
self.addEventListener('fetch', event => {
  if (event.request.method !== 'GET') return;
  event.respondWith(caches.match(event.request).then(cached => cached || fetch(event.request).then(response => {
    const copy = response.clone();
    caches.open(cacheName).then(cache => cache.put(event.request, copy));
    return response;
  })));
});
