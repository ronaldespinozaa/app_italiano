// Service worker de prototype/ — Fase 5 del roadmap (PWA instalable).
//
// Estrategia: cache-first con actualización en segundo plano ("stale-while
// -revalidate") para lo que sirve este mismo origen (index.html, manifest,
// icon). Todo lo que NO sea de este origen (los iframes de SoundCloud del
// módulo de Listening, sobre todo) se deja pasar sin tocar — no se cachea,
// no se intercepta; ese contenido sigue necesitando internet, ver
// docs/architecture.md decisión 7. No usa Workbox: es un solo archivo
// chico, más simple mantenerlo a mano que agregar una dependencia externa
// y romper el "sin build step" del resto del prototipo.
//
// El sufijo de CACHE_NAME es un hash corto de app.wasm, no un número que
// alguien tenga que acordarse de subir a mano: .github/workflows/
// wasm-build.yml lo recalcula y comitea en el mismo paso que recompila
// app.wasm (ver "Bump CACHE_NAME en sw.js" ahí). Por qué hace falta esto y
// no alcanza con el stale-while-revalidate de fetch() de abajo: cambiar
// CACHE_NAME cambia los BYTES de este archivo, y el navegador solo dispara
// el evento 'install' (que repuebla TODO el app shell de una — cache.
// addAll() — antes de que el SW nuevo tome control) cuando detecta que
// sw.js cambió byte a byte. Sin el bump, un app.wasm nuevo se actualiza
// recién de a poco vía el fetch en segundo plano de cada request individual
// — sin garantía de que quede sincronizado con el index.html que ya se
// actualizó. Hallazgo real de code review (2026-08-17, ver migration-log/
// log.json): CI ya había recompilado app.wasm dos veces sin que nada acá
// lo reflejara. Si compilás a mano con build.ps1 en vez de dejar que CI lo
// haga, actualizá este hash vos mismo (sha256sum prototype/app.wasm | cut
// -c1-10) o el próximo push a wasmapp/*.go lo va a pisar de todas formas.
const CACHE_NAME = 'italian-club-v2-a45f1c1ff5';
const APP_SHELL = [
  './', './index.html', './manifest.json', './icon.svg',
  './app.wasm', './wasm_exec.js',
];

self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME).then((cache) => cache.addAll(APP_SHELL))
  );
  self.skipWaiting();
});

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys().then((keys) =>
      Promise.all(keys.filter((k) => k !== CACHE_NAME).map((k) => caches.delete(k)))
    )
  );
  self.clients.claim();
});

self.addEventListener('fetch', (event) => {
  const req = event.request;
  const url = new URL(req.url);

  // Solo intervenimos GET del mismo origen — todo lo demás (SoundCloud,
  // Google Analytics si lo hubiera, etc.) pasa directo, sin caché propia.
  if (req.method !== 'GET' || url.origin !== self.location.origin) return;

  event.respondWith(
    caches.match(req).then((cached) => {
      const network = fetch(req)
        .then((resp) => {
          if (resp && resp.status === 200) {
            const clone = resp.clone();
            caches.open(CACHE_NAME).then((cache) => cache.put(req, clone));
          }
          return resp;
        })
        .catch(() => cached); // sin red y sin match previo: deja que falle normal
      return cached || network;
    })
  );
});
