// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { vi } from "vitest";

/** A fake MessagePort that delivers each message to its peer at once. */
export class FakePort {
  peer: FakePort | null = null;
  onmessage: ((event: { data: unknown }) => void) | null = null;
  closed = false;

  postMessage(data: unknown): void {
    if (!this.closed) this.peer?.onmessage?.({ data });
  }

  /** Closes both ends, as closing either end of a real channel does. */
  close(): void {
    this.closed = true;
    if (this.peer != null) this.peer.closed = true;
  }
}

/**
 * Stubs OffscreenCanvas and MessageChannel, which jsdom lacks, for the bitmap handoff.
 * Each new offscreen canvas takes the next of contexts, then a 2D context that draws
 * nothing. A transfer to a bitmap returns the canvas size. The caller must call
 * `vi.unstubAllGlobals` after the test.
 * @returns the first port of each channel made, in order.
 */
export const stubHandoff = (contexts: unknown[]): FakePort[] => {
  const ports: FakePort[] = [];
  vi.stubGlobal(
    "MessageChannel",
    class {
      port1 = new FakePort();
      port2 = new FakePort();

      constructor() {
        this.port1.peer = this.port2;
        this.port2.peer = this.port1;
        ports.push(this.port1);
      }
    },
  );
  vi.stubGlobal(
    "OffscreenCanvas",
    class extends EventTarget {
      private readonly context: unknown;

      constructor(
        public width: number,
        public height: number,
      ) {
        super();
        this.context =
          contexts.length > 0
            ? contexts.shift()
            : { canvas: this, drawImage: () => {} };
      }

      getContext(): unknown {
        return this.context;
      }

      transferToImageBitmap(): unknown {
        return { width: this.width, height: this.height };
      }
    },
  );
  return ports;
};
