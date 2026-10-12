// Chuck phone service worker (TS-08.R73). It only displays pushed
// attention notifications, routes a tap to the matching card, and keeps the
// app shell available. It never caches API responses: the Mac is the only
// source of truth, and stale data must look stale (FS-20.R23).
const SHELL = "chuck-shell-v1";

self.addEventListener("install", () => self.skipWaiting());
self.addEventListener("activate", (event) => {
  event.waitUntil(
    caches
      .keys()
      .then((keys) => Promise.all(keys.filter((key) => key !== SHELL).map((key) => caches.delete(key))))
      .then(() => self.clients.claim()),
  );
});

self.addEventListener("fetch", (event) => {
  const url = new URL(event.request.url);
  if (event.request.method !== "GET" || url.origin !== self.location.origin || url.pathname.startsWith("/api/")) return;
  if (event.request.mode === "navigate") {
    event.respondWith(
      fetch(event.request)
        .then((response) => {
          const copy = response.clone();
          caches.open(SHELL).then((cache) => cache.put("/", copy));
          return response;
        })
        .catch(() => caches.match("/")),
    );
    return;
  }
  if (url.pathname.startsWith("/assets/")) {
    event.respondWith(
      caches.match(event.request).then(async (hit) => {
        if (hit && isValidAssetResponse(hit, event.request.url)) return hit;
        const cache = await caches.open(SHELL);
        if (hit) await cache.delete(event.request);
        const response = await fetch(event.request);
        if (isValidAssetResponse(response, event.request.url)) await cache.put(event.request, response.clone());
        return response;
      }),
    );
  }
});

function isValidAssetResponse(response, requestURL) {
  if (!response.ok) return false;
  const type = response.headers.get("Content-Type")?.split(";", 1)[0].trim().toLowerCase();
  const path = new URL(requestURL).pathname;
  if (path.endsWith(".css")) return type === "text/css";
  if (path.endsWith(".js") || path.endsWith(".mjs")) {
    return type === "application/javascript" || type === "text/javascript" || type === "application/x-javascript";
  }
  return type !== "text/html" && type !== "application/json";
}

// Notifications carry no actions: every decision happens inside the app
// (FS-20.R21). A shared tag replaces rather than stacks (FS-20.R20).
self.addEventListener("push", (event) => {
  let payload = {};
  try {
    payload = event.data ? event.data.json() : {};
  } catch {
    payload = {};
  }
  const title = typeof payload.title === "string" ? payload.title : "Chuck";
  event.waitUntil(
    self.registration.showNotification(title, {
      body: typeof payload.body === "string" ? payload.body : "",
      tag: typeof payload.tag === "string" ? payload.tag : "chuck",
      renotify: true,
      icon: "/remote-icon.svg",
      data: { url: typeof payload.url === "string" && payload.url.startsWith("/") ? payload.url : "/" },
    }),
  );
});

self.addEventListener("notificationclick", (event) => {
  event.notification.close();
  const target = event.notification.data?.url || "/";
  event.waitUntil(
    self.clients.matchAll({ type: "window", includeUncontrolled: true }).then((clients) => {
      for (const client of clients) {
        if ("focus" in client) {
          client.postMessage({ kind: "open", url: target });
          return client.focus();
        }
      }
      return self.clients.openWindow(target);
    }),
  );
});
