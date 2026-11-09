export type ComposerSelection = {
  start: number;
  end: number;
  direction: HTMLTextAreaElement["selectionDirection"];
};

export type ComposerSnapshot = { text: string; selection: ComposerSelection };
type Deletion = { direction: "backward" | "forward"; at: number; hold?: object };
type Step = {
  offset: number;
  removed: string;
  inserted: string;
  before: ComposerSelection;
  after: ComposerSelection;
};

const MAX_STEPS = 100;
const MAX_TEXT_BYTES = 4 * 1024 * 1024;
const DELETE_WINDOW_MS = 500;

export function caret(pos: number): ComposerSelection {
  return { start: pos, end: pos, direction: "none" };
}

export function readComposer(el: HTMLTextAreaElement): ComposerSnapshot {
  return {
    text: el.value,
    selection: { start: el.selectionStart, end: el.selectionEnd, direction: el.selectionDirection },
  };
}

export function sameSelection(a: ComposerSelection, b: ComposerSelection) {
  return a.start === b.start && a.end === b.end && a.direction === b.direction;
}

function difference(before: ComposerSnapshot, after: ComposerSnapshot, deletion?: Deletion): Step {
  const oldText = before.text;
  const newText = after.text;
  const size = oldText.length - newText.length;
  // Anchor native deletions to their caret, even in a run of identical characters.
  if (deletion && size > 0) {
    const start = deletion.direction === "backward" ? before.selection.start - size : before.selection.start;
    if (start >= 0 && start === after.selection.start && newText === oldText.slice(0, start) + oldText.slice(start + size)) {
      return { offset: start, removed: oldText.slice(start, start + size), inserted: "", before: before.selection, after: after.selection };
    }
  }
  let offset = 0;
  let oldEnd = oldText.length;
  let newEnd = newText.length;
  while (offset < oldEnd && offset < newEnd && oldText[offset] === newText[offset]) offset++;
  while (oldEnd > offset && newEnd > offset && oldText[oldEnd - 1] === newText[newEnd - 1]) {
    oldEnd--;
    newEnd--;
  }
  return { offset, removed: oldText.slice(offset, oldEnd), inserted: newText.slice(offset, newEnd), before: before.selection, after: after.selection };
}

// Bound the UTF-16 text retained in both stacks. Replaying moves a step
// between them without adding any recorded text.
const bytes = (step: Step) => 2 * (step.removed.length + step.inserted.length);

export class ComposerHistory {
  private undo: Step[] = [];
  private redo: Step[] = [];
  private size = 0;
  private deletion?: Deletion;

  closeGroup() {
    this.deletion = undefined;
  }

  reset() {
    this.undo = [];
    this.redo = [];
    this.size = 0;
    this.closeGroup();
  }

  record(before: ComposerSnapshot, after: ComposerSnapshot, deletion?: Deletion) {
    if (before.text === after.text) return;
    if (before.selection.start !== before.selection.end || after.selection.start !== after.selection.end) deletion = undefined;
    let step = difference(before, after, deletion);
    const previous = this.undo[this.undo.length - 1];
    const group = this.deletion;
    if (step.inserted || step.offset !== after.selection.start) deletion = undefined;
    if (deletion && group && previous && deletion.direction === group.direction &&
        (deletion.at - group.at <= DELETE_WINDOW_MS || (deletion.hold && deletion.hold === group.hold)) &&
        sameSelection(before.selection, previous.after)) {
      const backward = deletion.direction === "backward";
      if (backward ? step.offset + step.removed.length === previous.offset : step.offset === previous.offset) {
        this.undo.pop();
        this.size -= bytes(previous);
        step = {
          ...step,
          removed: backward ? step.removed + previous.removed : previous.removed + step.removed,
          before: previous.before,
        };
      }
    }
    if (bytes(step) > MAX_TEXT_BYTES) {
      this.reset();
      return;
    }
    for (const undone of this.redo) this.size -= bytes(undone);
    this.redo = [];
    this.undo.push(step);
    this.size += bytes(step);
    while (this.undo.length > MAX_STEPS || this.size > MAX_TEXT_BYTES) {
      this.size -= bytes(this.undo.shift()!);
    }
    this.deletion = deletion;
  }

  replay(text: string, redo: boolean): ComposerSnapshot | undefined {
    this.closeGroup();
    const from = redo ? this.redo : this.undo;
    const to = redo ? this.undo : this.redo;
    const step = from.pop();
    if (!step) return;
    const { offset, inserted, removed, before, after } = step;
    const replace = redo ? removed : inserted;
    to.push(step);
    return {
      text: text.slice(0, offset) + (redo ? inserted : removed) + text.slice(offset + replace.length),
      selection: redo ? after : before,
    };
  }
}
