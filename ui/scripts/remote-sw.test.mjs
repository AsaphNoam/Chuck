import assert from "node:assert/strict";
import fs from "node:fs";
import test from "node:test";
import vm from "node:vm";

const source = fs.readFileSync(new URL("../public/remote-sw.js", import.meta.url), "utf8");

// Mobile stylesheet recovery. Execute the shipped worker, with real
// Request/Response objects and an in-memory Cache Storage boundary.
for (const failure of [
  { status: 404, type: "application/json", body: '{"error":{"code":"remote_route_not_available"}}' },
  { status: 200, type: "text/html", body: "<!doctype html><title>Chuck</title>" },
  { status: 200, type: "", body: "body { color: red; }" },
]) {
  test(`stylesheet recovers after ${failure.status} ${failure.type}`, async () => {
    const handlers = new Map();
    const stored = new Map();
    let networkCalls = 0;
    vm.runInNewContext(source, {
      URL,
      self: {
        location: { origin: "https://chuck.example.ts.net" },
        addEventListener: (name, handler) => handlers.set(name, handler),
      },
      caches: {
        match: async (request) => stored.get(request.url)?.clone(),
        open: async () => ({
          delete: async (request) => stored.delete(request.url),
          put: async (request, response) => stored.set(request.url, response),
        }),
      },
      fetch: async () => {
        networkCalls++;
        return networkCalls === 1
          ? new Response(failure.body, { status: failure.status, headers: { "Content-Type": failure.type } })
          : new Response("body { color: red; }", { headers: { "Content-Type": "text/css" } });
      },
    });
    const request = new Request("https://chuck.example.ts.net/assets/remote-example.css");
    const read = async () => {
      let response;
      handlers.get("fetch")({ request, respondWith: (promise) => { response = promise; } });
      return response;
    };
    await read();
    // Allow the worker's unawaited cache write to complete before reloading.
    await new Promise((resolve) => setImmediate(resolve));
    const recovered = await read();
    assert.equal(recovered.headers.get("Content-Type"), "text/css");
    assert.equal(await recovered.text(), "body { color: red; }");
    assert.equal(networkCalls, 2, "a failed stylesheet must be retried after recovery");
  });
}

for (const type of ["application/json", "text/html"]) test(`stylesheet repairs a cached ${type} response`, async () => {
  const handlers = new Map();
  const stored = new Map([
    ["https://chuck.example.ts.net/assets/remote-example.css", new Response("old", {
      status: type === "application/json" ? 404 : 200,
      headers: { "Content-Type": type },
    })],
  ]);
  let networkCalls = 0;
  vm.runInNewContext(source, {
    URL,
    self: {
      location: { origin: "https://chuck.example.ts.net" },
      addEventListener: (name, handler) => handlers.set(name, handler),
    },
    caches: {
      match: async (request) => stored.get(request.url)?.clone(),
      open: async () => ({
        delete: async (request) => stored.delete(request.url),
        put: async (request, response) => stored.set(request.url, response),
      }),
    },
    fetch: async () => {
      networkCalls++;
      return new Response("body { color: blue; }", { headers: { "Content-Type": "text/css" } });
    },
  });
  const request = new Request("https://chuck.example.ts.net/assets/remote-example.css");
  let response;
  handlers.get("fetch")({ request, respondWith: (promise) => { response = promise; } });
  const recovered = await response;
  assert.equal(recovered.headers.get("Content-Type"), "text/css");
  assert.equal(await recovered.text(), "body { color: blue; }");
  assert.equal(networkCalls, 1);
});
