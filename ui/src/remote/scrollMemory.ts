// Per-route scroll positions for the page session (FS-20.R54, TS-08.R125).
// Bounded and never persisted: a reload starts fresh.

export type Place = { y: number } | { row: string; offset: number } | { newest: true };

const MAX_PLACES = 50;
const places = new Map<string, Place>();

export function rememberPlace(path: string, place: Place) {
  places.delete(path);
  places.set(path, place);
  if (places.size > MAX_PLACES) places.delete(places.keys().next().value as string);
}

export function recallPlace(path: string): Place | undefined {
  return places.get(path);
}

export function forgetPlaces() {
  places.clear();
}
