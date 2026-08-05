// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { alamos } from "@synnaxlabs/alamos";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Context as TelemContext, CONTEXT_KEY } from "@/telem/aether/context";
import { CompoundFactory } from "@/telem/aether/factory";
import { type StringSourceSpec } from "@/telem/aether/telem";
import { TestFactory } from "@/telem/aether/test/factory";
import { source, stringSourceSpec, type TestSource } from "@/telem/aether/test/source";
import { StringValue } from "@/vis/stringValue/aether/stringValue";

const PATH = ["test-string-value"];

interface Harness {
  leaf: StringValue;
  update: (state?: Record<string, unknown>) => void;
}

const createLeaf = (telem: StringSourceSpec, stalenessTimeout?: number): Harness => {
  const leaf = new StringValue({
    path: PATH,
    type: StringValue.TYPE,
    sender: { send: vi.fn() },
    instrumentation: alamos.Instrumentation.NOOP,
    parentCtxValues: new Map([
      [CONTEXT_KEY, new TelemContext(new CompoundFactory([new TestFactory()]))],
    ]),
  });
  let initialized = false;
  // The main thread echoes the leaf's own state back on every update, so an update
  // carries the current value and staleness unless the test overrides them.
  const update = (state: Record<string, unknown> = {}): void => {
    const base = initialized ? leaf.state : { telem, stalenessTimeout };
    initialized = true;
    leaf._updateState({
      path: PATH,
      type: StringValue.TYPE,
      state: { ...base, ...state },
      create: () => leaf,
    });
  };
  update();
  return { leaf, update };
};

describe("StringValue", () => {
  let src: TestSource<string>;

  beforeEach(() => {
    vi.useFakeTimers();
    src = source("");
  });

  afterEach(() => {
    vi.useRealTimers();
    src.cleanup();
  });

  it("should publish the source's value on the first update", () => {
    src.setValue("IDLE");
    const { leaf } = createLeaf(stringSourceSpec(src));
    expect(leaf.state.value).toBe("IDLE");
    expect(leaf.state.stale).toBe(false);
  });

  it("should publish a value received from the source", () => {
    const { leaf } = createLeaf(stringSourceSpec(src));
    expect(leaf.state.value).toBe("");
    src.setValue("ARMED");
    expect(leaf.state.value).toBe("ARMED");
  });

  it("should go stale once the timeout elapses without a value", () => {
    const { leaf } = createLeaf(stringSourceSpec(src), 2);
    src.setValue("ARMED");
    expect(leaf.state.stale).toBe(false);
    vi.advanceTimersByTime(1999);
    expect(leaf.state.stale).toBe(false);
    vi.advanceTimersByTime(1);
    expect(leaf.state.stale).toBe(true);
  });

  it("should restart the countdown on every received value", () => {
    const { leaf } = createLeaf(stringSourceSpec(src), 2);
    src.setValue("ARMED");
    vi.advanceTimersByTime(1500);
    src.setValue("FIRING");
    vi.advanceTimersByTime(1500);
    expect(leaf.state.stale).toBe(false);
    vi.advanceTimersByTime(500);
    expect(leaf.state.stale).toBe(true);
  });

  it("should clear staleness when a value arrives after going stale", () => {
    const { leaf } = createLeaf(stringSourceSpec(src), 2);
    src.setValue("ARMED");
    vi.advanceTimersByTime(2000);
    expect(leaf.state.stale).toBe(true);
    src.setValue("FIRING");
    expect(leaf.state.stale).toBe(false);
    expect(leaf.state.value).toBe("FIRING");
  });

  // A prop change re-reads the source without a new sample behind it, so it must not
  // present a stale value as fresh.
  it("should stay stale across an update that does not carry a value", () => {
    const { leaf, update } = createLeaf(stringSourceSpec(src), 2);
    src.setValue("ARMED");
    vi.advanceTimersByTime(2000);
    expect(leaf.state.stale).toBe(true);
    update({ stalenessTimeout: 10 });
    expect(leaf.state.stale).toBe(true);
  });

  it("should follow the new source when the telem spec changes", () => {
    const { leaf, update } = createLeaf(stringSourceSpec(src));
    src.setValue("FIRST");
    const next = source("SECOND");
    update({ telem: stringSourceSpec(next) });
    expect(leaf.state.value).toBe("SECOND");
    src.setValue("IGNORED");
    expect(leaf.state.value).toBe("SECOND");
    next.cleanup();
  });

  it("should stop the countdown and release the source when deleted", () => {
    const { leaf } = createLeaf(stringSourceSpec(src), 2);
    const cleanup = vi.spyOn(src, "cleanup");
    src.setValue("ARMED");
    leaf._delete(PATH);
    vi.advanceTimersByTime(2000);
    expect(leaf.state.stale).toBe(false);
    expect(cleanup).toHaveBeenCalledTimes(1);
  });
});
