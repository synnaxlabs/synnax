// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Channel } from "@/channel/client";
import type { Key, Payload, RetrieveRequest, Where } from "@/channel/types.gen";
import type { group } from "@/group";
import { query } from "@/query";

/** Joins the names on a path. Matches the separator of the Core. */
const SEPARATOR = ".";

/** @returns whether the where, or a where inside it, filters by path. */
const filtersPath = (where?: Where): boolean =>
  where != null &&
  (where.path != null ||
    where.and?.some(filtersPath) === true ||
    where.or?.some(filtersPath) === true ||
    filtersPath(where.not));

interface Folder {
  name: string;
  parent?: string;
}

const folderOf = ({ name, parent }: group.Group): Folder => ({
  name,
  parent: parent?.type === "group" ? parent.key : undefined,
});

export interface PathsConfig {
  /** The table of the channels. */
  channels: query.Table<Key, Channel>;
  /** The groups, whose table holds the folders on each path. */
  groups: group.Client;
  /** Receives the errors of the folder reads that no caller waits on. */
  onError: (error: Error) => void;
}

/**
 * Builds the path of a channel from its name and the cached folders, as the Core builds
 * it, so a rename or a move changes the path with no channel fetch.
 */
export class Paths {
  /**
   * Composes again the members of an answer with paths when their paths change, and
   * fetches again an answer that filters by path when a channel folder changes.
   */
  readonly triggers: Array<query.Trigger<RetrieveRequest, Key>>;
  private readonly cfg: PathsConfig;
  /**
   * The name and parent of each channel folder that Paths has seen, so a set tells a
   * rename or a move apart from a write that keeps both.
   */
  private readonly folders = new Map<group.Key, Folder>();
  /** Counts the renames and moves of the folders that Paths has seen. */
  private moves = 0;
  /** Counts the moves and each change to a channel folder that Paths has not seen. */
  private pathMatches = 0;
  /**
   * The channels whose path the Core left absent, each with the folder it sat in then.
   * The Core leaves it absent when the caller cannot retrieve a folder on it.
   */
  private readonly hidden = new Map<Key, group.Key>();
  /** The folders whose ancestors a read is fetching. */
  private readonly reading = new Set<group.Key>();

  constructor(cfg: PathsConfig) {
    this.cfg = cfg;
    const { channels, groups } = cfg;
    // A folder that the Core deletes holds nothing, so a delete changes no path. This
    // subscriber counts before the triggers' subscribers read the counts.
    groups.store.subscribe((event) => {
      if (event.variant === "delete") {
        this.folders.delete(event.key);
        return;
      }
      if (event.value.scope !== "channel") return;
      const before = this.folders.get(event.key);
      const after = folderOf(event.value);
      this.folders.set(event.key, after);
      // A folder that Paths has not seen is either new and empty, or one that a write
      // changed before the client read it. Only the second can change what a path
      // matches, and only through a folder that the client did not cache.
      if (before == null) {
        if (query.isChange(event)) this.pathMatches++;
        return;
      }
      if (before.name === after.name && before.parent === after.parent) return;
      this.moves++;
      this.pathMatches++;
    });
    channels.subscribe((event) => {
      if (event.variant === "delete") this.hidden.delete(event.key);
    });
    const composed: query.Trigger<RetrieveRequest, Key> = {
      attach: (params, onEvent, members) => {
        if (params.includePath !== true) return () => {};
        let folders = new Set<group.Key>();
        let moved = new Set<Key>();
        let timer: ReturnType<typeof setTimeout> | undefined;
        // Runs once every write of a stream frame applies, since a frame can make a
        // folder and move a channel into it, and an answer admits a member after this
        // sees the member's write.
        const flush = (): void => {
          timer = undefined;
          const changed: Key[] = [];
          const lacking: Channel[] = [];
          for (const key of members()) {
            const ch = channels.get(key);
            if (ch == null) continue;
            // An answer composes a member again for its own write, so only a folder
            // write marks it changed.
            const passes = this.passes(ch.group, folders);
            if (passes) changed.push(key);
            if ((passes || moved.has(key)) && this.get(ch) == null) lacking.push(ch);
          }
          folders = new Set();
          moved = new Set();
          if (changed.length > 0) onEvent(changed);
          if (lacking.length > 0) this.complete(lacking, onEvent);
        };
        const schedule = (): void => {
          timer ??= setTimeout(flush, 0);
        };
        const offGroups = groups.store.subscribe((event) => {
          if (event.variant === "delete" || event.value.scope !== "channel") return;
          folders.add(event.key);
          schedule();
        });
        // The commit echo of this client's own move is a set again, so a move that the
        // Core had not applied when the answer fetched reads its folders after it.
        const offChannels = channels.subscribe((event) => {
          if (event.variant !== "set" || !query.isChange(event)) return;
          moved.add(event.key);
          schedule();
        });
        return () => {
          clearTimeout(timer);
          offGroups();
          offChannels();
        };
      },
      version: () => this.moves,
      contentOnly: true,
    };
    const filtered: query.Trigger<RetrieveRequest, Key> = {
      attach: (params, onEvent) => {
        if (!filtersPath(params.where)) return () => {};
        let seen = this.pathMatches;
        return groups.store.subscribe(() => {
          if (this.pathMatches === seen) return;
          seen = this.pathMatches;
          onEvent("refetch");
        });
      },
      version: () => this.pathMatches,
    };
    this.triggers = [composed, filtered];
  }

  /**
   * @returns the path of the channel, or undefined when a folder on it is not cached or
   * the Core left it absent.
   */
  get(ch: Pick<Payload, "key" | "name" | "group">): string | undefined {
    if (ch.group == null) return ch.name;
    if (this.hidden.get(ch.key) === ch.group) return undefined;
    return this.build(ch);
  }

  /**
   * Reads the folders that the client lacks or holds old on each path that the Core
   * sent, and keeps each path that the Core left absent absent.
   */
  async fill(payloads: Payload[]): Promise<void> {
    const off = new Set<group.Key>();
    for (const p of payloads) {
      if (p.group == null) continue;
      if (p.path == null) {
        this.hidden.set(p.key, p.group);
        continue;
      }
      this.hidden.delete(p.key);
      if (this.build(p) !== p.path) off.add(p.group);
    }
    if (off.size === 0) return;
    const { store } = this.cfg.groups;
    const keys = [...off];
    keys.forEach((key) => this.reading.add(key));
    try {
      await Promise.all([
        store.retrieve(keys.filter((key) => !store.has(key))),
        this.cfg.groups.retrieveAncestors(keys),
      ]);
      // A cached folder on a path that still differs is old, as on a client with no
      // change stream.
      const old = new Set<group.Key>();
      for (const p of payloads)
        if (p.group != null && off.has(p.group) && this.build(p) !== p.path)
          old.add(p.group);
      if (old.size > 0) await store.retrieve([...old], { refresh: true });
    } finally {
      keys.forEach((key) => this.reading.delete(key));
    }
  }

  private build({
    name,
    group: g,
  }: Pick<Payload, "name" | "group">): string | undefined {
    if (g == null) return name;
    const folders = this.cfg.groups.getCachedPath(g);
    return folders == null ? undefined : [...folders, name].join(SEPARATOR);
  }

  /** @returns whether the cached chain of folders up from start passes a folder. */
  private passes(start: group.Key | undefined, folders: Set<group.Key>): boolean {
    if (start == null || folders.size === 0) return false;
    return this.cfg.groups
      .getCachedChain(start)
      .groups.some(({ key }) => folders.has(key));
  }

  /**
   * Reads the folders above each channel whose folder is cached, and fetches the answer
   * again for a channel whose folder is not, since only the Core knows whether the
   * caller can retrieve it.
   */
  private complete(
    lacking: Channel[],
    onEvent: (result: Key[] | "refetch") => void,
  ): void {
    const keys = new Set<group.Key>();
    for (const { key, group: g } of lacking) {
      if (g == null || this.hidden.get(key) === g || this.reading.has(g)) continue;
      if (!this.cfg.groups.store.has(g)) return onEvent("refetch");
      keys.add(g);
    }
    if (keys.size === 0) return;
    const reads = [...keys];
    reads.forEach((key) => this.reading.add(key));
    this.cfg.groups
      .retrieveAncestors(reads)
      .catch((exc: unknown) => {
        this.cfg.onError(
          new Error("failed to read the folders of channel paths", { cause: exc }),
        );
      })
      .finally(() => reads.forEach((key) => this.reading.delete(key)));
  }
}
