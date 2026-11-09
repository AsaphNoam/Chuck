# Desktop chat composer editing

**State:** Finished
**Why:** Operator request on 2026-11-09 and follow-up approval to include undoing autocomplete.
**Relevant requirements:** [FS-03.R113–R116](../specs/features/FS-03-chat.md#214-composer-editing),
[TS-08.R158–R160](../specs/tech/TS-08-frontend-presentation.md#219-desktop-composer-editing),
INV §1, INV §2, INV §3, INV §10, INV §16, INV §17.

## Outcome

Undo a burst of deletions or a picker insertion with one shortcut. Continue, nest and unnest
numbered items while writing a prompt.

## Included work

Full desktop agent chat and expanded dashboard chat share their existing plain-text composer.
Keep the approved scope to deletion undo, numbered lists and sublists, and undoable file/command
autocomplete. Bullet lists, multi-item indentation, a new paragraph shortcut, phone and Think
Tank composition are out of scope.

Keep the native textarea with an in-memory edit history. The existing picker writes its
replacement as controlled text. In the repository's Playwright 1.63.0 Chromium, Cmd+Z restored
two adjacent native Backspaces in an isolated textarea, but did not reverse a scripted
`value` assignment or `setRangeText` replacement. This probe did not exercise the product
or other browsers. The [Input Events specification](https://www.w3.org/TR/input-events-2/#interface-InputEvent)
defines edit types and history events. The [HTML range replacement API](https://html.spec.whatwg.org/multipage/form-control-infrastructure.html#dom-textarea-setrangetext)
does not define an undo transaction, and the [legacy command draft](https://w3c.github.io/editing/docs/execCommand/)
documents inconsistent implementation. A single composer history covers both native edits and
scripted replacements without relying on legacy commands. Use [CommonMark list indentation](https://spec.commonmark.org/0.31.2/#list-items)
for generated sublists.

## How we will know it works

Exercise [FS-03.A89–A91](../specs/features/FS-03-chat.md#5-acceptance-criteria) with focused edit
and composer tests and a rendered desktop journey. Confirm held-key undo/redo, focus exit,
input-method composition, picker replacement and caret restoration, exact submitted text, and
draft ownership while switching agents or awaiting send completion.

## Waiting on

None.

## Delivered verification

- `make check-specs`, `cd ui && npm test` (1,193 tests in 115 files), and `cd ui && npm run build`
  passed for the completed source and specification changes.
- Rendered the full agent chat and expanded dashboard chat in Playwright Chromium against fixture
  API responses. Verified held Backspace across a newline, timed and caret-separated Delete groups,
  backward selection replay, file and command replacement, Escape followed by either Tab direction,
  numbered continuation/nesting/promotion, plain and fenced newlines, literal multiline input,
  exact prompt and transcript text, reload and agent isolation, and newer edits during delayed send.
  Chromium DevTools drove composition and a composing key event exercised the Send guard.
  Local harness and results are at `/private/tmp/chuck-composer-browser.mjs` and
  `/private/tmp/chuck-composer-browser.log`.
