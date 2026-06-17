// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";
import { crdt } from "@synnaxlabs/x";

// SEED_REPLICA is the replica the server uses to seed a document from persisted text.
// Clients bootstrap from the server snapshot and must never author with this id, so a
// client's edits never collide with the seed.
const SEED_REPLICA = 1;

// randomReplica returns a replica id in [2, 2^32) for a client editing session, avoiding
// the reserved seed replica.
const randomReplica = (): number =>
  SEED_REPLICA + 1 + Math.floor(Math.random() * (0xffffffff - SEED_REPLICA - 1));

const codePoints = (s: string): string[] => Array.from(s);

export interface Diff {
  index: number;
  deleteCount: number;
  insert: string;
}

/** diff computes the single contiguous change that turns oldStr into newStr, measured in
 * code points: the longest common prefix and suffix are stripped and what remains in the
 * middle is the deletion (from oldStr) and insertion (from newStr). */
export const diff = (oldStr: string, newStr: string): Diff => {
  const o = codePoints(oldStr);
  const n = codePoints(newStr);
  const max = Math.min(o.length, n.length);
  let prefix = 0;
  while (prefix < max && o[prefix] === n[prefix]) prefix++;
  let suffix = 0;
  while (
    suffix < o.length - prefix &&
    suffix < n.length - prefix &&
    o[o.length - 1 - suffix] === n[n.length - 1 - suffix]
  )
    suffix++;
  return {
    index: prefix,
    deleteCount: o.length - prefix - suffix,
    insert: n.slice(prefix, n.length - suffix).join(""),
  };
};

const toActions = (inserts: crdt.Insert[], deletes: crdt.Delete[]): arc.Action[] => [
  ...deletes.map((op) => arc.deleteChar({ op })),
  ...inserts.map((op) => arc.insertChar({ op })),
];

/** CollabText binds a replicated text document to a plain-string editing surface. It
 * bootstraps from a server snapshot so it shares the server's id space, translates
 * whole-value edits into CRDT operations to broadcast, applies remote operations, and
 * always exposes the materialized string. It does not know about the transport or the
 * editor: callers feed it new values and remote operations and read back the value. */
export class CollabText {
  private readonly doc: crdt.Text;

  private constructor(doc: crdt.Text) {
    this.doc = doc;
  }

  /** bootstrap creates a document from a server snapshot so the client's edits compose
   * with every other editor's in the same id space. */
  static bootstrap(snapshot: {
    inserts: crdt.Insert[];
    deletes: crdt.Delete[];
  }): CollabText {
    const doc = new crdt.Text(randomReplica());
    doc.load(snapshot);
    return new CollabText(doc);
  }

  /** value returns the current materialized text. */
  value(): string {
    return this.doc.toString();
  }

  /** edit reconciles the document to next and returns the operations that describe the
   * change, to be broadcast to the other editors. It returns an empty array when next is
   * already the current value. */
  edit(next: string): arc.Action[] {
    const { index, deleteCount, insert } = diff(this.value(), next);
    if (deleteCount === 0 && insert.length === 0) return [];
    const deletes = deleteCount > 0 ? this.doc.delete(index, deleteCount) : [];
    const inserts = insert.length > 0 ? this.doc.insert(index, insert) : [];
    return toActions(inserts, deletes);
  }

  /** applyRemote integrates operations produced by other editors. */
  applyRemote(actions: arc.Action[]): void {
    for (const a of actions)
      if (a.type === "insert_char") this.doc.applyInsert(a.insertChar.op);
      else if (a.type === "delete_char") this.doc.applyDelete(a.deleteChar.op);
  }
}
