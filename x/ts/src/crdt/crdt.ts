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

const ROOT_ID: ID = { replica: 0, counter: 0 };

const isRoot = (id: ID): boolean => id.replica === 0 && id.counter === 0;

const idLess = (a: ID, b: ID): boolean =>
  a.replica !== b.replica ? a.replica < b.replica : a.counter < b.counter;

const idKey = (id: ID): string => `${id.replica}:${id.counter}`;

// TO_STRING_CHUNK bounds how many code points are passed to a single
// String.fromCodePoint call. Spreading an unbounded array of arguments overflows the
// call stack for large documents, so the materialization is built in chunks.
const TO_STRING_CHUNK = 8192;

/** Element is a run of characters authored by one replica with contiguous counters.
 * The in-order traversal of the run tree, skipping deleted characters, yields the
 * materialized document. Invariants: left children anchor to the first character,
 * right children anchor to the last, and interior characters never carry children; an
 * anchor into the interior splits the run first. */
interface Element {
  /** id identifies the first character; the character at offset i has counter
   * id.counter+i. */
  id: ID;
  chars: number[];
  /** deleted marks tombstoned characters. It is null when none are; otherwise its
   * length always equals chars.length. */
  deleted: boolean[] | null;
  /** dead is the number of tombstoned characters in this run. */
  dead: number;
  left: Element[];
  right: Element[];
}

const lastID = (e: Element): ID => ({
  replica: e.id.replica,
  counter: e.id.counter + e.chars.length - 1,
});

const charID = (e: Element, i: number): ID => ({
  replica: e.id.replica,
  counter: e.id.counter + i,
});

/** kill tombstones the character at offset off, reporting whether it was live. */
const kill = (e: Element, off: number): boolean => {
  e.deleted ??= new Array<boolean>(e.chars.length).fill(false);
  if (e.deleted[off]) return false;
  e.deleted[off] = true;
  e.dead += 1;
  return true;
};

/** hasRight reports whether the character at offset off anchors anything on its right:
 * a successor within the run, or, for the last character, a right child. */
const hasRight = (e: Element, off: number): boolean =>
  off < e.chars.length - 1 || e.right.length > 0;

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
  private counter = 0;
  private readonly root: Element;
  // index maps each replica to its runs ordered by starting counter, so any character
  // id resolves to its containing run by binary search.
  private readonly index = new Map<number, Element[]>();
  private readonly tombstones = new Set<string>();
  private pending: Insert[] = [];
  private order: Element[] = [];
  private dirty = true;
  // lastPlaced is the run most recently created or extended: the likely target of the
  // next sequential insert, letting a typed or seeded run extend without an id lookup.
  private lastPlaced: Element | null = null;
  // stringCache caches the materialized string; null when stale.
  private stringCache: string | null = null;
  // live is the number of non-deleted characters, maintained incrementally so len is
  // constant time.
  private live = 0;

  constructor(replica: number) {
    this.replica = replica;
    this.root = { id: ROOT_ID, chars: [], deleted: null, dead: 0, left: [], right: [] };
  }

  /** replicaID returns the id of the replica that owns this document. */
  replicaID(): number {
    return this.replica;
  }

  /** snapshot captures the full current state of the document as the operations that
   * reconstruct it when applied to an empty document, in an order where every
   * operation's origin precedes it. It is used to bootstrap a replica joining an
   * in-progress session. Deleted characters are included as both an insert and a delete
   * so anchoring is preserved. */
  snapshot(): { inserts: Insert[]; deletes: Delete[] } {
    const inserts: Insert[] = [];
    const deletes: Delete[] = [];
    const stack: Array<{ node: Element; origin: ID; side: spatial.XLocation }> = [
      { node: this.root, origin: ROOT_ID, side: "right" },
    ];
    while (stack.length > 0) {
      const frame = stack.pop();
      if (frame == null) break;
      const { node, origin, side } = frame;
      let first = origin;
      let last = origin;
      if (node !== this.root) {
        let prev = origin;
        let prevSide = side;
        for (let i = 0; i < node.chars.length; i++) {
          const id = charID(node, i);
          inserts.push({ id, origin: prev, side: prevSide, char: node.chars[i] });
          if (node.deleted?.[i] === true) deletes.push({ id });
          prev = id;
          prevSide = "right";
        }
        first = node.id;
        last = lastID(node);
      }
      for (let i = node.right.length - 1; i >= 0; i--)
        stack.push({ node: node.right[i], origin: last, side: "right" });
      for (let i = node.left.length - 1; i >= 0; i--)
        stack.push({ node: node.left[i], origin: first, side: "left" });
    }
    return { inserts, deletes };
  }

  /** load applies a snapshot to the document. It is intended for a freshly created
   * document; the local replica's edits remain attributed to its own replica id and do
   * not collide with the snapshot's operations. */
  load(snapshot: { inserts: Insert[]; deletes: Delete[] }): void {
    this.applyInsert(...snapshot.inserts);
    this.applyDelete(...snapshot.deletes);
  }

  // markDirty invalidates every derived cache after a structural mutation: one that
  // creates, splits, or attaches a run and so changes the traversal.
  private markDirty(): void {
    this.dirty = true;
    this.stringCache = null;
  }

  // markStale invalidates only the materialized string, for mutations that change a
  // run's content in place (an appended or tombstoned character) without changing the
  // traversal.
  private markStale(): void {
    this.stringCache = null;
  }

  private rebuild(): void {
    if (!this.dirty) return;
    this.order.length = 0;
    this.walk();
    this.dirty = false;
  }

  /** walk performs the in-order traversal of the run tree into order, visiting each
   * node's left children, then the node, then its right children. The traversal is
   * iterative with an explicit stack because a heavily interleaved document forms a
   * tree as deep as its run count, which could overflow the call stack under
   * recursion. */
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

  // findRun returns the run containing id and the character's offset within it.
  private findRun(id: ID): [Element, number] | null {
    const runs = this.index.get(id.replica);
    if (runs == null) return null;
    let lo = 0;
    let hi = runs.length;
    while (lo < hi) {
      const mid = (lo + hi) >> 1;
      if (runs[mid].id.counter <= id.counter) lo = mid + 1;
      else hi = mid;
    }
    if (lo === 0) return null;
    const e = runs[lo - 1];
    const off = id.counter - e.id.counter;
    if (off < e.chars.length) return [e, off];
    return null;
  }

  // indexInsert records a new run in the replica index, keeping runs counter-ordered.
  private indexInsert(e: Element): void {
    let runs = this.index.get(e.id.replica);
    if (runs == null) {
      runs = [];
      this.index.set(e.id.replica, runs);
    }
    let lo = 0;
    let hi = runs.length;
    while (lo < hi) {
      const mid = (lo + hi) >> 1;
      if (runs[mid].id.counter < e.id.counter) lo = mid + 1;
      else hi = mid;
    }
    runs.splice(lo, 0, e);
  }

  /** splitAfter splits e after character offset k (0 <= k < length-1) and returns the
   * new tail run. The tail inherits e's right children, since they anchor to e's old
   * last character, and becomes e's sole right child. */
  private splitAfter(e: Element, k: number): Element {
    const tail: Element = {
      id: charID(e, k + 1),
      chars: e.chars.slice(k + 1),
      deleted: null,
      dead: 0,
      left: [],
      right: e.right,
    };
    e.chars.length = k + 1;
    if (e.deleted != null) {
      tail.deleted = e.deleted.slice(k + 1);
      e.deleted.length = k + 1;
      for (const d of tail.deleted) if (d) tail.dead += 1;
      e.dead -= tail.dead;
    }
    e.right = [tail];
    this.indexInsert(tail);
    return tail;
  }

  // charAt returns the run and character offset of the live character at index.
  private charAt(index: number): [Element, number] | null {
    if (index < 0 || index >= this.live) return null;
    this.rebuild();
    let seen = 0;
    for (const e of this.order) {
      const liveHere = e.chars.length - e.dead;
      if (index >= seen + liveHere) {
        seen += liveHere;
        continue;
      }
      let k = index - seen;
      if (e.dead === 0) return [e, k];
      for (let i = 0; i < e.chars.length; i++) {
        if (e.deleted?.[i] === true) continue;
        if (k === 0) return [e, i];
        k -= 1;
      }
    }
    return null;
  }

  /** len returns the number of live characters in the document. */
  len(): number {
    return this.live;
  }

  /** toString materializes the document into its current string value. The result is
   * cached and reused until the next mutation. */
  toString(): string {
    if (this.stringCache != null) return this.stringCache;
    this.rebuild();
    let out = "";
    let buf: number[] = [];
    const flush = (): void => {
      out += String.fromCodePoint(...buf);
      buf = [];
    };
    for (const e of this.order)
      for (let i = 0; i < e.chars.length; i++) {
        if (e.dead > 0 && e.deleted?.[i] === true) continue;
        buf.push(e.chars[i]);
        if (buf.length >= TO_STRING_CHUNK) flush();
      }

    if (buf.length > 0) flush();
    this.stringCache = out;
    return out;
  }

  /** indexToID returns the id of the character at the given live index, or null when
   * the index is out of range. */
  indexToID(index: number): ID | null {
    const found = this.charAt(index);
    if (found == null) return null;
    return charID(found[0], found[1]);
  }

  /** insert inserts text at the given live index and returns the operations that
   * describe the edit. The operations are applied to this document before they are
   * returned, and must be broadcast to other replicas for them to converge. index is
   * measured in code points and may equal len() to append. */
  insert(index: number, text: string): Insert[] {
    const chars = Array.from(text);
    if (chars.length === 0) return [];
    let left = this.root;
    let leftOff = -1;
    if (index > 0) {
      const at = this.charAt(index - 1) ?? this.charAt(this.live - 1);
      if (at != null) [left, leftOff] = at;
    }
    const rightAt = this.charAt(index);
    const rightID = rightAt != null ? charID(rightAt[0], rightAt[1]) : null;
    const ops: Insert[] = [];
    for (const ch of chars) {
      let origin: ID;
      let side: spatial.XLocation;
      if (rightID != null && hasRight(left, leftOff)) {
        origin = rightID;
        side = "left";
      } else {
        origin = left === this.root ? ROOT_ID : charID(left, leftOff);
        side = "right";
      }
      this.counter += 1;
      const op: Insert = {
        id: { replica: this.replica, counter: this.counter },
        origin,
        side,
        char: ch.codePointAt(0) ?? 0,
      };
      const placed = this.place(op);
      if (placed != null) [left, leftOff] = placed;
      ops.push(op);
    }
    return ops;
  }

  /** delete removes length characters starting at the given live index and returns the
   * operations that describe the edit. The operations are applied to this document
   * before they are returned. */
  delete(index: number, length: number): Delete[] {
    if (length <= 0 || index < 0 || index >= this.live) return [];
    this.rebuild();
    const end = Math.min(index + length, this.live);
    const ids: ID[] = [];
    let seen = 0;
    for (const e of this.order) {
      const liveHere = e.chars.length - e.dead;
      if (seen + liveHere <= index) {
        seen += liveHere;
        continue;
      }
      for (let i = 0; i < e.chars.length; i++) {
        if (e.deleted?.[i] === true) continue;
        if (seen >= index && seen < end) ids.push(charID(e, i));
        seen += 1;
      }
      if (seen >= end) break;
    }
    const ops: Delete[] = [];
    for (const id of ids) {
      const op: Delete = { id };
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
      if (this.place(op) != null) {
        if (this.pending.length > 0) this.drain();
      } else this.pending.push(op);
  }

  /** applyDelete integrates delete operations produced by other replicas. A delete
   * whose character has not yet been seen is recorded so the character is tombstoned on
   * arrival. */
  applyDelete(...ops: Delete[]): void {
    for (const op of ops) {
      const found = this.findRun(op.id);
      if (found != null) {
        const [e, off] = found;
        if (kill(e, off)) {
          this.live -= 1;
          this.markStale();
        }
        continue;
      }
      this.tombstones.add(idKey(op.id));
    }
  }

  /** extendable returns the run op contiguously appends to, or null: the run last
   * touched by place, when op's origin is its last character, replica and counter
   * continue it, and nothing anchors after it. Such an op cannot be a duplicate: were
   * its counter already placed, the run would have been split or given a right child,
   * and the check would fail. */
  private extendable(op: Insert): Element | null {
    const e = this.lastPlaced;
    if (e == null || op.side !== "right" || e.right.length > 0) return null;
    const last = lastID(e);
    const ok =
      op.id.replica === e.id.replica &&
      op.origin.replica === last.replica &&
      op.origin.counter === last.counter &&
      op.id.counter === e.id.counter + e.chars.length;
    return ok ? e : null;
  }

  /** extend appends op's character to the run extendable approved. */
  private extend(e: Element, op: Insert): [Element, number] {
    e.chars.push(op.char);
    e.deleted?.push(false);
    const off = e.chars.length - 1;
    if (this.tombstones.size > 0 && this.tombstones.delete(idKey(op.id))) kill(e, off);
    else this.live += 1;
    this.markStale();
    return [e, off];
  }

  /** place attaches an insert to the run tree, returning the run and offset holding
   * the character. It returns null without buffering when the operation's origin is not
   * yet present. */
  private place(op: Insert): [Element, number] | null {
    const run = this.extendable(op);
    if (run != null) return this.extend(run, op);
    const existing = this.findRun(op.id);
    if (existing != null) return existing;
    let origin = this.root;
    let originOff = -1;
    if (!isRoot(op.origin)) {
      const found = this.findRun(op.origin);
      if (found == null) return null;
      [origin, originOff] = found;
    }
    const e: Element = {
      id: op.id,
      chars: [op.char],
      deleted: null,
      dead: 0,
      left: [],
      right: [],
    };
    if (this.tombstones.delete(idKey(op.id))) kill(e, 0);
    else this.live += 1;
    if (op.side === "left") {
      if (originOff > 0) origin = this.splitAfter(origin, originOff - 1);
      sortedInsert(origin.left, e);
    } else {
      if (originOff >= 0 && originOff < origin.chars.length - 1)
        this.splitAfter(origin, originOff);
      sortedInsert(origin.right, e);
    }
    this.indexInsert(e);
    this.lastPlaced = e;
    this.markDirty();
    return [e, 0];
  }

  private drain(): void {
    for (;;) {
      let progressed = false;
      const remaining: Insert[] = [];
      for (const op of this.pending) {
        if (this.findRun(op.id) != null) continue;
        if (!isRoot(op.origin) && this.findRun(op.origin) == null) {
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
