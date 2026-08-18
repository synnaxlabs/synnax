// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { telemTest } from "@/telem/aether/test";
import { renderAether } from "@/testutil/renderAether";
import { StringValue } from "@/vis/stringValue/aether/stringValue";

// Mounts a StringValue under the real provider stack with a string telem source
// registered before mount, so the telem TestFactory resolves it on the first
// afterUpdate.
const setup = (source: telemTest.TestSource<string>, stalenessTimeout?: number) =>
  renderAether(StringValue, {
    state: {
      telem: telemTest.stringSourceSpec(source),
      ...(stalenessTimeout != null ? { stalenessTimeout } : {}),
    },
  });

describe("StringValue", () => {
  let src: telemTest.TestSource<string>;

  beforeEach(() => {
    vi.useFakeTimers();
    src = telemTest.source("");
  });

  afterEach(() => {
    vi.useRealTimers();
    src.cleanup();
  });

  it("should publish the source's value on the first update", () => {
    src.setValue("IDLE");
    const h = setup(src);
    expect(h.state.value).toBe("IDLE");
    expect(h.state.stale).toBe(false);
  });

  it("should publish a value received from the source", () => {
    const h = setup(src);
    expect(h.state.value).toBe("");
    src.setValue("ARMED");
    expect(h.state.value).toBe("ARMED");
  });

  it("should go stale once the timeout elapses without a value", () => {
    const h = setup(src, 2);
    src.setValue("ARMED");
    expect(h.state.stale).toBe(false);
    vi.advanceTimersByTime(1999);
    expect(h.state.stale).toBe(false);
    vi.advanceTimersByTime(1);
    expect(h.state.stale).toBe(true);
  });

  it("should restart the countdown on every received value", () => {
    const h = setup(src, 2);
    src.setValue("ARMED");
    vi.advanceTimersByTime(1500);
    src.setValue("FIRING");
    vi.advanceTimersByTime(1500);
    expect(h.state.stale).toBe(false);
    vi.advanceTimersByTime(500);
    expect(h.state.stale).toBe(true);
  });

  it("should clear staleness when a value arrives after going stale", () => {
    const h = setup(src, 2);
    src.setValue("ARMED");
    vi.advanceTimersByTime(2000);
    expect(h.state.stale).toBe(true);
    src.setValue("FIRING");
    expect(h.state.stale).toBe(false);
    expect(h.state.value).toBe("FIRING");
  });

  // A prop change re-reads the source without a new sample behind it, so it must not
  // present a stale value as fresh.
  it("should stay stale across an update that does not carry a value", () => {
    const h = setup(src, 2);
    src.setValue("ARMED");
    vi.advanceTimersByTime(2000);
    expect(h.state.stale).toBe(true);
    h.setState((p) => ({ ...p, stalenessTimeout: 10 }));
    expect(h.state.stale).toBe(true);
  });

  it("should follow the new source when the telem spec changes", () => {
    const h = setup(src);
    src.setValue("FIRST");
    const next = telemTest.source("SECOND");
    h.setState((p) => ({ ...p, telem: telemTest.stringSourceSpec(next) }));
    expect(h.state.value).toBe("SECOND");
    src.setValue("IGNORED");
    expect(h.state.value).toBe("SECOND");
    next.cleanup();
  });

  it("should stop the countdown and release the source when deleted", () => {
    const h = setup(src, 2);
    const cleanup = vi.spyOn(src, "cleanup");
    src.setValue("ARMED");
    h.unmount();
    vi.advanceTimersByTime(2000);
    expect(h.state.stale).toBe(false);
    expect(cleanup).toHaveBeenCalledTimes(1);
  });
});
