# Roomier file reading with adjustable panes and a focus modal

**State:** Waiting to start
**Why:** Operator request and approval, 2026-10-17: files lack space, Markdown escapes its box,
and file width should flex relative to chat and annotations. The operator confirmed the combined
split/focus scope and required avoiding accidental Chinese characters in authored interface copy.
**Relevant requirements:** FS-03.R92–R97/A71–A75; TS-08.R127–R132; existing FS-03.R52/R54/R64/R65,
FS-12.R61, FS-13.R18–R22/R25, TS-06.R3–R6; INV §1/§2/§8/§10/§13/§16/§17.

## Outcome

Read a file comfortably beside an agent conversation, resize file/chat/annotation space, or use
Focus file for sustained reading and annotation. Return without losing reading location, loaded
snapshot, composer draft, annotation instructions or delivery state.

## Included work

- Contained Markdown with ordinary chat-sized text and document spacing, wrapping inline code and
  locally scrolling wide technical blocks. Shared styling also reaches the existing room viewer.
- Full live/archived agent-page dividers and a Radix focus modal using the same file controller,
  shared annotation tray and existing read/selection/delivery paths. English authored copy and
  repository icons; original file/path/excerpt Unicode remains intact.
- Page-lifetime requested proportions, bounded width allocation, explicit focus/return and
  preserved context. No new browser preference, stored file content, endpoint or read authority.
- No editor, file tabs/history, outline/search subsystem, new phone reader or room-layout redesign.
  Dashboard links retain their navigation to the full agent reader.

The implementation uses the existing workspace/portal seams, installed Radix Dialog and shared
Markdown renderer. `FileViewer` currently owns read/timestamp/mode state locally and `AnnotationTray`
owns transient delivery state locally; lift those owners only enough to survive presentation
changes. `PointerContextMenu` currently portals to `document.body`; focus requires a scoped
destination inside dialog content so menus remain interactive and Escape closes the inner layer
first. These are verified source seams, not assumed library limitations. The resize exception
to TS-08.R53/R57/R112 is narrowly defined by R128, not permission for general measurement logic.

## How we will know it works

FS-03.A71–A75 cover long inline code/paths, wide tables/fences/diagrams, header fit, both dividers,
minimum/reset/collapse/overlay behavior, focus trapping/return and nested menus, unchanged read
counts and draft/delivery state, source-line navigation, loading/refusal/partial reads and English
UI copy. Exercise live and archived sources at 1024px, 1440px and wider in all three desktop
appearances, plus shared-room styling and dashboard/phone compatibility. Include New task launch,
failed delivery and focus transitions while delivery is pending. Use focused component/state
checks plus actual browser geometry and task journeys; implementation closure follows TS-06 and
workflow §2. These product checks remain owed until implementation.

Design evidence: incumbent browser at 2137px allocated 640px to the file and about 1418px to chat;
the 639px file body had 716px scroll width from inline code. Rendered text was 13.76px with normal
line height; source confirmed capped widths and document spacing limited to assistant messages.
The local composition sketch is `.chuck-dev/design/file-reader.html` (not production code or an
acceptance receipt). Official references informed the split/focus and Rendered/Source patterns:
[VS Code layout](https://code.visualstudio.com/docs/configure/custom-layout),
[Markdown preview](https://code.visualstudio.com/docs/languages/markdown),
[Obsidian tabs](https://obsidian.md/help/tabs),
[GitHub file views](https://docs.github.com/en/repositories/working-with-files/using-files/viewing-and-understanding-files).

## Waiting on

None. Behavior and page-lifetime sizing were approved; implementation has not started.
