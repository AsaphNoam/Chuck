import { describe, expect, it } from "vitest";
import { allocateReader, fileShareFor, MIN_CHAT, MIN_FILE, MIN_TRAY } from "./readerLayout";

const initial = { fileShare: null, trayWidth: null };

// FS-03.A72, TS-08.R128 — requested proportions stay separate from the widths
// that fit, minimums hold, and three columns dock only when all of them fit.
describe("allocateReader", () => {
  it("gives the file half of the space left after a docked tray", () => {
    const allocation = allocateReader({ available: 1600, stage: 1600, file: true, tray: "expanded", request: initial });
    expect(allocation.tray).toBe("docked");
    expect(allocation.trayWidth?.now).toBe(460);
    expect(allocation.file?.now).toBe(570);
    expect(allocation.chat).toBe(570);
  });

  it("starts the tray at 30% of the stage below the 460px cap", () => {
    const allocation = allocateReader({ available: 1200, stage: 1200, file: false, tray: "expanded", request: initial });
    expect(allocation.trayWidth?.now).toBe(360);
    expect(allocation.chat).toBe(840);
  });

  it("clamps requests to the useful minimums without rewriting them", () => {
    const request = { fileShare: 0.95, trayWidth: 2000 };
    const allocation = allocateReader({ available: 1400, stage: 1400, file: true, tray: "expanded", request });
    expect(allocation.trayWidth?.now).toBe(1400 - MIN_FILE - MIN_CHAT);
    expect(allocation.file?.now).toBe(MIN_FILE);
    expect(allocation.chat).toBe(MIN_CHAT);
    expect(request).toEqual({ fileShare: 0.95, trayWidth: 2000 });

    const small = allocateReader({ available: 1400, stage: 1400, file: true, tray: "none", request: { fileShare: 0.01, trayWidth: null } });
    expect(small.file?.now).toBe(MIN_FILE);
  });

  it("overlays the tray when three useful columns do not fit, leaving the file its split", () => {
    const tight = MIN_FILE + MIN_CHAT + MIN_TRAY - 1;
    const allocation = allocateReader({ available: tight, stage: tight, file: true, tray: "expanded", request: initial });
    expect(allocation.tray).toBe("overlay");
    expect(allocation.trayWidth).toBeNull();
    expect((allocation.file?.now ?? 0) + allocation.chat).toBe(tight);
    expect(allocateReader({ available: tight + 1, stage: tight + 1, file: true, tray: "expanded", request: initial }).tray).toBe("docked");
  });

  it("keeps the chosen split when the tray collapses or goes away", () => {
    const request = { fileShare: 0.6, trayWidth: null };
    const docked = allocateReader({ available: 1600, stage: 1600, file: true, tray: "expanded", request });
    const collapsed = allocateReader({ available: 1550, stage: 1600, file: true, tray: "collapsed", request });
    const none = allocateReader({ available: 1600, stage: 1600, file: true, tray: "none", request });
    for (const allocation of [docked, collapsed, none]) {
      const share = (allocation.file?.now ?? 0) / ((allocation.file?.now ?? 0) + allocation.chat);
      expect(share).toBeCloseTo(0.6, 2);
    }
    expect(collapsed.tray).toBe("collapsed");
    expect(none.file!.now).toBeGreaterThan(docked.file!.now);
  });

  it("stays adjacent and scales both minimums below them", () => {
    const allocation = allocateReader({ available: 600, stage: 600, file: true, tray: "none", request: { fileShare: 0.9, trayWidth: null } });
    expect(allocation.file?.now).toBe(Math.round(600 * MIN_FILE / (MIN_FILE + MIN_CHAT)));
    expect((allocation.file?.now ?? 0) + allocation.chat).toBe(600);
  });

  it("turns a dragged width into a share of the file and conversation space", () => {
    const allocation = allocateReader({ available: 1000, stage: 1000, file: true, tray: "none", request: initial });
    expect(fileShareFor(600, allocation)).toBeCloseTo(0.6);
    expect(fileShareFor(5000, allocation)).toBeCloseTo((1000 - MIN_CHAT) / 1000);
  });
});
