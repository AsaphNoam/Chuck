// The agent header's Copy thread identity (FS-03.R62), the transcript's Copy
// selection (FS-03.R63) and message Copy (FS-01.R62) share one clipboard path so
// the availability check and the "Copy failed" wording cannot drift apart
// (INV §2). Failures surface as a toast rather than silently doing nothing
// (INV §8). The result says whether the write succeeded, for callers that show
// copied feedback.
export function copyText(text: string, pushError: (title: string, body: string) => void): Promise<boolean> {
  if (!navigator.clipboard) {
    pushError("Copy failed", "Clipboard access is not available in this browser.");
    return Promise.resolve(false);
  }
  return navigator.clipboard.writeText(text).then(() => true, (error) => {
    pushError("Copy failed", error instanceof Error ? error.message : String(error));
    return false;
  });
}
