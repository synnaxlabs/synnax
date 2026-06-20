// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc, type framer, type Synnax } from "@synnaxlabs/client";
import { type destructor, id } from "@synnaxlabs/x";

import { CollabText, type Diff } from "@/arc/collab";

/** CollabSession ties the CRDT document for an arc to the cluster transport. It
 * bootstraps the document from the server snapshot, streams remote operations off the
 * arc signals channel, and dispatches local edits. It is editor-agnostic: callers read
 * value(), push whole new values through edit(), and subscribe() to learn when remote
 * edits change the value. */
export class CollabSession {
  private readonly client: Synnax;
  private readonly key: arc.Key;
  private readonly text: CollabText;
  private readonly streamer: framer.Streamer;
  private readonly listeners = new Set<(value: string) => void>();
  // outstanding holds the dispatch keys of local edits awaiting their broadcast echo, so
  // the echo is recognized as already-applied and skipped rather than re-integrated.
  private readonly outstanding = new Set<string>();
  private closed = false;

  private constructor(
    client: Synnax,
    key: arc.Key,
    text: CollabText,
    streamer: framer.Streamer,
  ) {
    this.client = client;
    this.key = key;
    this.text = text;
    this.streamer = streamer;
  }

  /** open bootstraps the document from the server and begins streaming remote edits. */
  static async open(client: Synnax, key: arc.Key): Promise<CollabSession> {
    const retrieved = await client.arcs.retrieve({ key });
    const text = CollabText.bootstrap(retrieved.text.doc);
    const streamer = await client.openStreamer({ channels: [arc.SET_CHANNEL_NAME] });
    const session = new CollabSession(client, key, text, streamer);
    void session.listen();
    return session;
  }

  /** value returns the current materialized text. */
  value(): string {
    return this.text.value();
  }

  /** subscribe registers a callback invoked with the new value whenever a remote edit
   * changes the document. The returned destructor removes the callback. */
  subscribe(cb: (value: string) => void): destructor.Destructor {
    this.listeners.add(cb);
    return () => this.listeners.delete(cb);
  }

  /** applyChanges applies a batch of local editor changes to the document and broadcasts
   * the resulting operations. The edits are applied immediately and optimistically; the
   * server echo is recognized by its dispatch key and skipped. */
  applyChanges(changes: Diff[]): void {
    const actions = this.text.applyChanges(changes);
    if (actions.length === 0) return;
    const dispatchKey = id.create();
    this.outstanding.add(dispatchKey);
    this.client.arcs.dispatch(this.key, dispatchKey, actions).catch((err: unknown) => {
      this.outstanding.delete(dispatchKey);
      console.error("failed to dispatch arc edit", err);
    });
  }

  private async listen(): Promise<void> {
    try {
      for await (const frame of this.streamer) {
        if (this.closed) break;
        const series = frame.get(arc.SET_CHANNEL_NAME);
        if (series.length === 0) continue;
        let changed = false;
        for (const scoped of series.parseJSON(arc.scopedActionZ)) {
          if (scoped.key !== this.key) continue;
          // Skip the echo of a local edit: it is already applied to the document.
          if (this.outstanding.delete(scoped.dispatchKey)) continue;
          this.text.applyRemote(scoped.actions);
          changed = true;
        }
        if (changed) {
          const value = this.text.value();
          for (const cb of this.listeners) cb(value);
        }
      }
    } catch (err) {
      if (!this.closed) console.error("arc collaborative session stream closed", err);
    }
  }

  /** close stops streaming and releases the underlying transport. */
  close(): void {
    this.closed = true;
    this.listeners.clear();
    this.streamer.close();
  }
}
