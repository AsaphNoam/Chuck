// FS-20.A31: pulling up at the newest message reloads; other touches do not.
import React from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { PULL_THRESHOLD, usePullUpRefresh } from "./pullUpRefresh";

function Harness({ onRefresh }: { onRefresh: () => void }) {
  const state = usePullUpRefresh(true, onRefresh);
  return <div><p data-testid="state">{state}</p><p data-testid="message">Newest reply</p><textarea aria-label="Message" /></div>;
}

const point = (clientY: number) => ({ touches: [{ clientY }] });
const pull = (target: Element, distance: number, fingers = 1) => {
  const touches = (y: number) => ({ touches: Array.from({ length: fingers }, () => ({ clientY: y })) });
  fireEvent.touchStart(target, touches(600));
  fireEvent.touchMove(target, touches(600 - distance));
  return () => fireEvent.touchEnd(target, { touches: [] });
};

describe("usePullUpRefresh", () => {
  afterEach(() => {
    cleanup();
    Object.defineProperty(document.documentElement, "scrollHeight", { configurable: true, value: 0 });
  });

  it("hints during the pull and reloads once released past the threshold at the newest message", () => {
    const onRefresh = vi.fn();
    render(<Harness onRefresh={onRefresh} />);
    const message = screen.getByTestId("message");
    fireEvent.touchStart(message, point(600));
    fireEvent.touchMove(message, point(580));
    expect(screen.getByTestId("state").textContent).toBe("pulling");
    fireEvent.touchMove(message, point(600 - PULL_THRESHOLD));
    expect(screen.getByTestId("state").textContent).toBe("armed");
    fireEvent.touchEnd(message, { touches: [] });
    expect(onRefresh).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId("state").textContent).toBe("refreshing");
  });

  it("does not reload for a short pull, a scrolled-up page, the text field, a focused field or two fingers", () => {
    const onRefresh = vi.fn();
    render(<Harness onRefresh={onRefresh} />);
    const message = screen.getByTestId("message");
    const field = screen.getByLabelText("Message");

    pull(message, PULL_THRESHOLD - 1)();
    expect(screen.getByTestId("state").textContent).toBe("idle");

    pull(field, PULL_THRESHOLD * 2)();
    pull(message, PULL_THRESHOLD * 2, 2)();

    act(() => field.focus());
    pull(message, PULL_THRESHOLD * 2)();
    act(() => field.blur());

    Object.defineProperty(document.documentElement, "scrollHeight", { configurable: true, value: window.innerHeight + 500 });
    pull(message, PULL_THRESHOLD * 2)();

    expect(onRefresh).not.toHaveBeenCalled();
    expect(screen.getByTestId("state").textContent).toBe("idle");
  });

  it("does not reload when the pull ends after new content moved the newest message away", () => {
    const onRefresh = vi.fn();
    render(<Harness onRefresh={onRefresh} />);
    const release = pull(screen.getByTestId("message"), PULL_THRESHOLD * 2);
    Object.defineProperty(document.documentElement, "scrollHeight", { configurable: true, value: window.innerHeight + 500 });
    release();
    expect(onRefresh).not.toHaveBeenCalled();
  });
});
