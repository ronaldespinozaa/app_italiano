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
// Subir CACHE_NAME (v1 -> v2...) en cada cambio de fondo grande fuerza a
// los clientes viejos a descartar su caché en el próximo activate.
const CACHE_NAME = 'italian-club-v2'; // v2: agrega app.wasm/wasm_exec.js (motor de ejercicios)
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
