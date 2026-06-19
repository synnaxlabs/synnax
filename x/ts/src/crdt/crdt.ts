// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Delete, type ID, type Insert } from "@/crdt/types.gen";
import { type spatial } from "@/spatial";

const ROOT_ID: ID = { replica: 0, counter: 0n };

const isRoot = (id: ID): boolean => id.replica === 0 && id.counter === 0n;

const idLess = (a: ID, b: ID): boolean =>
  a.replica !== b.replica ? a.replica < b.replica : a.counter < b.counter;

const idKey = (id: ID): string => `${id.replica}:${id.counter}`;

// TO_STRING_CHUNK bounds how many code points are passed to a single
// String.fromCodePoint call. Spreading an unbounded array of arguments overflows the
// call stack for large documents, so the materialization is built in chunks.
const TO_STRING_CHUNK = 8192;

interface Element {
  id: ID;
  char: number;
  deleted: boolean;
  left: Element[];
  right: Element[];
}

const sortedInsert = (children: Element[], e: Element): void => {
  let lo = 0;
  let hi = children.length;
  while (lo < hi) {
    const mid = (lo + hi) >> 1;
    if (idLess(children[mid].id, e.id)) lo = mid + 1;
    else hi = mid;
  }
  children.splice(lo, 0, e);
};

/** Text is a replicated text document owned by a single replica, identified at
 * construction; local edits are attributed to that replica. Text is not safe for
 * concurrent use. */
export class Text {
  private readonly replica: number;
  private counter = 0n;
  private readonly root: Element;
  private readonly elements = new Map<string, Element>();
  private readonly tombstones = new Set<string>();
  private pending: Insert[] = [];
  private order: Element[] = [];
  private dirty = true;

  constructor(replica: number) {
    this.replica = replica;
    this.root = {
      id: ROOT_ID,
      char: 0,
      deleted: false,
      left: [],
      right: [],
    };
  }

  /** replicaID returns the id of the replica that owns this document. */
  replicaID(): number {
    return this.replica;
  }

  private rebuild(): void {
    if (!this.dirty) return;
    this.order = [];
    this.walk();
    this.dirty = false;
  }

  /** walk performs the in-order traversal of the document tree into order, visiting
   * each node's left children, then the node, then its right children. The traversal
   * is iterative with an explicit stack because a document built by typing in a single
   * direction forms a tree as deep as it is long, which would overflow the call stack
   * under recursion. */
  private walk(): void {
    const stack: Array<{ node: Element; emit: boolean }> = [
      { node: this.root, emit: false },
    ];
    while (stack.length > 0) {
      const frame = stack.pop();
      if (frame == null) break;
      if (frame.emit) {
        this.order.push(frame.node);
        continue;
      }
      const { node } = frame;
      for (let i = node.right.length - 1; i >= 0; i--)
        stack.push({ node: node.right[i], emit: false });
      if (node !== this.root) stack.push({ node, emit: true });
      for (let i = node.left.length - 1; i >= 0; i--)
        stack.push({ node: node.left[i], emit: false });
    }
  }

  private visible(): Element[] {
    this.rebuild();
    return this.order.filter((e) => !e.deleted);
  }

  /** len returns the number of live characters in the document. */
  len(): number {
    return this.visible().length;
  }

  /** toString materializes the document into its current string value. */
  toString(): string {
    const vis = this.visible();
    let out = "";
    for (let i = 0; i < vis.length; i += TO_STRING_CHUNK) {
      const codes = vis.slice(i, i + TO_STRING_CHUNK).map((e) => e.char);
      out += String.fromCodePoint(...codes);
    }
    return out;
  }

  /** indexToID returns the id of the character at the given live index, or null when
   * the index is out of range. */
  indexToID(index: number): ID | null {
    const vis = this.visible();
    if (index < 0 || index >= vis.length) return null;
    return vis[index].id;
  }

  /** insert inserts text at the given live index and returns the operations that
   * describe the edit. The operations are applied to this document before they are
   * returned, and must be broadcast to other replicas for them to converge. index is
   * measured in code points and may equal len() to append. */
  insert(index: number, text: string): Insert[] {
    const chars = Array.from(text);
    if (chars.length === 0) return [];
    const vis = this.visible();
    let left: Element;
    if (index <= 0) left = this.root;
    else if (index - 1 < vis.length) left = vis[index - 1];
    else left = vis.length > 0 ? vis[vis.length - 1] : this.root;
    const right = index >= 0 && index < vis.length ? vis[index] : null;
    const ops: Insert[] = [];
    for (const ch of chars) {
      let origin: ID;
      let side: spatial.XLocation;
      if (right != null && left.right.length > 0) {
        origin = right.id;
        side = "left";
      } else {
        origin = left.id;
        side = "right";
      }
      this.counter += 1n;
      const op: Insert = {
        id: { replica: this.replica, counter: this.counter },
        origin,
        side,
        char: ch.codePointAt(0) ?? 0,
      };
      left = this.place(op) as Element;
      ops.push(op);
    }
    return ops;
  }

  /** delete removes length characters starting at the given live index and returns the
   * operations that describe the edit. The operations are applied to this document
   * before they are returned. */
  delete(index: number, length: number): Delete[] {
    if (length <= 0) return [];
    const vis = this.visible();
    const ops: Delete[] = [];
    for (let i = index; i < index + length && i < vis.length; i++) {
      const op: Delete = { id: vis[i].id };
      this.applyDelete(op);
      ops.push(op);
    }
    return ops;
  }

  /** applyInsert integrates insert operations produced by other replicas. Operations
   * may be supplied in any order and duplicates are ignored; operations whose origin
   * has not yet arrived are buffered and integrated once it does. */
  applyInsert(...ops: Insert[]): void {
    for (const op of ops)
      if (this.place(op) != null) this.drain();
      else this.pending.push(op);
  }

  /** applyDelete integrates delete operations produced by other replicas. A delete
   * whose character has not yet been seen is recorded so the character is tombstoned on
   * arrival. */
  applyDelete(...ops: Delete[]): void {
    for (const op of ops) {
      const e = this.elements.get(idKey(op.id));
      if (e != null) {
        if (!e.deleted) {
          e.deleted = true;
          this.dirty = true;
        }
        continue;
      }
      this.tombstones.add(idKey(op.id));
    }
  }

  private place(op: Insert): Element | null {
    const key = idKey(op.id);
    const existing = this.elements.get(key);
    if (existing != null) return existing;
    let origin = this.root;
    if (!isRoot(op.origin)) {
      const found = this.elements.get(idKey(op.origin));
      if (found == null) return null;
      origin = found;
    }
    const e: Element = {
      id: op.id,
      char: op.char,
      deleted: this.tombstones.delete(key),
      left: [],
      right: [],
    };
    if (op.side === "left") sortedInsert(origin.left, e);
    else sortedInsert(origin.right, e);
    this.elements.set(key, e);
    this.dirty = true;
    return e;
  }

  private drain(): void {
    for (;;) {
      let progressed = false;
      const remaining: Insert[] = [];
      for (const op of this.pending) {
        if (this.elements.has(idKey(op.id))) continue;
        if (!isRoot(op.origin) && !this.elements.has(idKey(op.origin))) {
          remaining.push(op);
          continue;
        }
        this.place(op);
        progressed = true;
      }
      this.pending = remaining;
      if (!progressed) return;
    }
  }
}
