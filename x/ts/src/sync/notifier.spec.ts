// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Notifier } from "@/sync/notifier";
import { TimeSpan } from "@/telem";

const settled = async <V>(promise: Promise<V>): Promise<boolean> => {
  let done = false;
  void promise.then(() => (done = true));
  await new Promise((resolve) => setTimeout(resolve, 5));
  return done;
};

describe("Notifier", () => {
  it("should release a waiter on notify", async () => {
    const notifier = new Notifier();
    const wait = notifier.wait(null);
    expect(await settled(wait)).toBe(false);
    notifier.notify();
    await expect(wait).resolves.toBeUndefined();
  });

  it("should return immediately when a notify is already pending", async () => {
    const notifier = new Notifier();
    notifier.notify();
    await expect(notifier.wait(null)).resolves.toBeUndefined();
  });

  it("should consume a pending notify exactly once", async () => {
    const notifier = new Notifier();
    notifier.notify();
    await notifier.wait(null);
    expect(await settled(notifier.wait(null))).toBe(false);
  });

  it("should release a waiter when its span elapses", async () => {
    const notifier = new Notifier();
    await expect(notifier.wait(TimeSpan.milliseconds(5))).resolves.toBeUndefined();
  });

  it("should wait indefinitely on a null span", async () => {
    const notifier = new Notifier();
    expect(await settled(notifier.wait(null))).toBe(false);
  });

  it("should wake every current waiter", async () => {
    const notifier = new Notifier();
    const waits = [notifier.wait(null), notifier.wait(null), notifier.wait(null)];
    notifier.notify();
    await expect(Promise.all(waits)).resolves.toHaveLength(3);
  });

  it("should not leave a notify pending when waiters consumed it", async () => {
    const notifier = new Notifier();
    const wait = notifier.wait(null);
    notifier.notify();
    await wait;
    expect(await settled(notifier.wait(null))).toBe(false);
  });

  it("should drop a timed-out waiter rather than wake it twice", async () => {
    const notifier = new Notifier();
    await notifier.wait(TimeSpan.milliseconds(1));
    notifier.notify();
    // the notify found no waiters, so it sticks for the next one
    await expect(notifier.wait(null)).resolves.toBeUndefined();
  });
});
