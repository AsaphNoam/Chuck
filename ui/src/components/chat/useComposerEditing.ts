import { useCallback, useLayoutEffect, useRef, useState, type ChangeEvent, type KeyboardEvent, type RefObject } from "react";
import { chatDraftRevision, setChatDraft } from "./drafts";
import { caret, ComposerHistory, readComposer, sameSelection, type ComposerSnapshot } from "./composerHistory";
import { editNumberedList } from "./composerLists";

const deletionDirection = (type: string) => type === "deleteContentBackward" ? "backward"
  : type === "deleteContentForward" ? "forward" : undefined;
const historyInput = (type: string) => type === "historyUndo" || type === "historyRedo";

export function useComposerEditing(
  agentId: string,
  text: string,
  setText: (value: string) => void,
  textarea: RefObject<HTMLTextAreaElement>,
) {
  const [history] = useState(() => new ComposerHistory());
  const beforeInput = useRef<ComposerSnapshot>();
  const observed = useRef<ComposerSnapshot>();
  const composition = useRef<ComposerSnapshot>();
  const pending = useRef<{ next: ComposerSnapshot; revision: number; agentId: string }>();
  const hold = useRef<{ key: string }>();
  const exitOnTab = useRef(false);

  const breakInteraction = () => {
    hold.current = undefined;
    exitOnTab.current = false;
    history.closeGroup();
  };

  const reset = useCallback(() => {
    history.reset();
    beforeInput.current = undefined;
    observed.current = undefined;
    composition.current = undefined;
    pending.current = undefined;
    hold.current = undefined;
    exitOnTab.current = false;
  }, [history]);

  const observe = (el: HTMLTextAreaElement) => {
    const snapshot = readComposer(el);
    if (observed.current && !sameSelection(observed.current.selection, snapshot.selection)) breakInteraction();
    observed.current = snapshot;
    return snapshot;
  };

  const commit = (next: ComposerSnapshot) => {
    setChatDraft(agentId, next.text);
    pending.current = { next, agentId, revision: chatDraftRevision(agentId) };
    setText(next.text);
  };

  // Read and publish the actual caret after React writes the controlled value.
  const flush = (sync: (el: HTMLTextAreaElement) => void) => {
    const update = pending.current;
    const el = textarea.current;
    if (!update || !el) return;
    pending.current = undefined;
    if (update.agentId !== agentId || update.revision !== chatDraftRevision(agentId) || el.value !== update.next.text) return;
    const { start, end, direction } = update.next.selection;
    if (!composition.current && !sameSelection(readComposer(el).selection, update.next.selection)) {
      el.setSelectionRange(start, end, direction);
    }
    observed.current = readComposer(el);
    if (!composition.current) sync(el);
  };

  const replay = (redo: boolean) => {
    beforeInput.current = undefined;
    breakInteraction();
    const next = history.replay(text, redo);
    if (next) commit(next);
  };

  // React's onBeforeInput is a synthetic text event and does not reliably
  // report deletion or browser-menu history. Observe the native event.
  useLayoutEffect(() => {
    const el = textarea.current;
    if (!el) return;
    const onBeforeInput = (event: InputEvent) => {
      if (historyInput(event.inputType) && event.cancelable && !composition.current) {
        event.preventDefault();
        replay(event.inputType === "historyRedo");
        return;
      }
      if (!composition.current) beforeInput.current = observe(el);
    };
    el.addEventListener("beforeinput", onBeforeInput);
    return () => el.removeEventListener("beforeinput", onBeforeInput);
  });

  const onChange = (event: ChangeEvent<HTMLTextAreaElement>) => {
    const input = event.nativeEvent as InputEvent;
    const type = input.inputType ?? "";
    if (historyInput(type)) {
      replay(type === "historyRedo");
      return;
    }
    const after = readComposer(event.currentTarget);
    const before = beforeInput.current ?? observed.current ?? { text, selection: caret(text.length) };
    beforeInput.current = undefined;
    observed.current = after;
    exitOnTab.current = false;
    if (input.isComposing) composition.current ??= before;
    if (!composition.current) {
      const direction = deletionDirection(type);
      const key = direction === "backward" ? "Backspace" : "Delete";
      history.record(before, after, direction ? {
        direction,
        at: event.timeStamp,
        hold: hold.current?.key === key ? hold.current : undefined,
      } : undefined);
    }
    commit(after);
  };

  const replace = (nextText: string, pos: number) => {
    const el = textarea.current;
    if (!el || composition.current) return;
    const before = readComposer(el);
    const next = { text: nextText, selection: caret(pos) };
    breakInteraction();
    if (before.text === nextText) {
      el.setSelectionRange(pos, pos);
      observed.current = readComposer(el);
      return;
    }
    history.record(before, next);
    commit(next);
  };

  const isComposing = (event: KeyboardEvent<HTMLTextAreaElement>) =>
    !!composition.current || event.nativeEvent.isComposing || event.keyCode === 229;

  const onKeyDown = (event: KeyboardEvent<HTMLTextAreaElement>) => {
    const { key, shiftKey, ctrlKey, metaKey, altKey } = event;
    const before = observe(event.currentTarget);
    if (key === "Shift" || key === "Control" || key === "Meta" || key === "Alt") return false;
    const bypass = exitOnTab.current;
    exitOnTab.current = false;
    if (key === "Backspace" || key === "Delete") {
      if (!event.repeat || hold.current?.key !== key) hold.current = { key };
      return false;
    }
    breakInteraction();
    if (!altKey && ctrlKey !== metaKey) {
      const lower = key.toLowerCase();
      if (lower === "z" || (ctrlKey && !shiftKey && lower === "y")) {
        event.preventDefault();
        replay(shiftKey || lower === "y");
        return true;
      }
    }
    if (altKey || metaKey || ctrlKey) return false;
    const command = key === "Enter" && shiftKey ? "continue"
      : key === "Tab" && !bypass ? (shiftKey ? "promote" : "indent")
      : key === "Escape" ? "exit" : undefined;
    if (!command) return false;
    const edit = editNumberedList(before, command);
    if (!edit) return false;
    event.preventDefault();
    if (command === "exit") exitOnTab.current = true;
    else {
      if (edit.intermediate) history.record(before, edit.intermediate);
      history.record(edit.intermediate ?? before, edit.next);
      commit(edit.next);
    }
    return true;
  };

  return {
    reset, flush, replace, onKeyDown, isComposing, breakInteraction,
    input: {
      onChange,
      onSelect: (event: React.SyntheticEvent<HTMLTextAreaElement>) => { observe(event.currentTarget); },
      onPointerDown: breakInteraction,
      onBlur: breakInteraction,
      onKeyUp: (event: KeyboardEvent<HTMLTextAreaElement>) => {
        if (hold.current?.key === event.key) hold.current = undefined;
      },
      onCompositionStart: (event: React.CompositionEvent<HTMLTextAreaElement>) => {
        breakInteraction();
        composition.current = readComposer(event.currentTarget);
      },
      onCompositionEnd: (event: React.CompositionEvent<HTMLTextAreaElement>) => {
        const before = composition.current;
        composition.current = undefined;
        if (!before) return;
        const after = readComposer(event.currentTarget);
        observed.current = after;
        history.record(before, after);
        if (after.text !== text) commit(after);
      },
    },
  };
}
