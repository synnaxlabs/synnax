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
 * it. A rename or a move of a channel or a folder changes the path with no channel
 * fetch.
 */
export class Paths {
  /**
   * Composes again the members of an answer that asks for paths when their paths
   * change, and reads the folders that a member's path lacks. Fetches again an answer
   * whose where filters by path when a channel folder is renamed or moved.
   */
  readonly triggers: Array<query.Trigger<RetrieveRequest, Key>>;
  private readonly cfg: PathsConfig;
  /** The name and parent of each channel folder in the group table. */
  private readonly folders = new Map<group.Key, Folder>();
  /** Counts the folder writes that can change a path. */
  private moves = 0;

  constructor(cfg: PathsConfig) {
    this.cfg = cfg;
    const { channels, groups } = cfg;
    // A folder that the Core deletes holds nothing, so a delete changes no path.
    groups.store.subscribe((event) => {
      if (event.variant === "delete") {
        this.folders.delete(event.key);
        return;
      }
      if (event.value.scope !== "channel") return;
      const before = this.folders.get(event.key);
      const after = folderOf(event.value);
      this.folders.set(event.key, after);
      // A folder new to the table is either a new, empty folder or one that a write
      // changed before the client read it, and only the second can change a path.
      if (
        before == null
          ? query.isChange(event)
          : before.name !== after.name || before.parent !== after.parent
      )
        this.moves++;
    });
    const composed: query.Trigger<RetrieveRequest, Key> = {
      attach: (params, onEvent, members) => {
        if (params.includePath !== true) return () => {};
        const offGroups = groups.store.subscribe((event) => {
          if (event.variant === "delete" || event.value.scope !== "channel") return;
          const changed: Key[] = [];
          const lacking: Channel[] = [];
          for (const key of members()) {
            const ch = channels.get(key);
            if (ch == null || !this.passes(ch.group, event.key)) continue;
            changed.push(key);
            if (this.get(ch) == null) lacking.push(ch);
          }
          if (changed.length > 0) onEvent(changed);
          if (lacking.length > 0) this.complete(lacking, onEvent);
        });
        const offChannels = channels.subscribe((event) => {
          if (event.variant !== "set" || !query.isChange(event)) return;
          if (!members().includes(event.key)) return;
          if (this.get(event.value) == null) this.complete([event.value], onEvent);
        });
        return () => {
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
        let moves = this.moves;
        return groups.store.subscribe(() => {
          if (this.moves === moves) return;
          moves = this.moves;
          onEvent("refetch");
        });
      },
      version: () => this.moves,
    };
    this.triggers = [composed, filtered];
  }

  /**
   * @returns the path of the channel, or undefined when a folder on it is not cached.
   */
  get({ name, group: g }: Pick<Payload, "name" | "group">): string | undefined {
    if (g == null) return name;
    const folders = this.cfg.groups.getCachedPath(g);
    return folders == null ? undefined : [...folders, name].join(SEPARATOR);
  }

  /** @returns whether the chain of folders up from start passes the folder. */
  private passes(start: group.Key | undefined, folder: group.Key): boolean {
    const passed = new Set<group.Key>();
    for (let at = start; at != null && !passed.has(at);) {
      if (at === folder) return true;
      passed.add(at);
      const g = this.cfg.groups.store.get(at);
      at = g?.parent?.type === "group" ? g.parent.key : undefined;
    }
    return false;
  }

  /**
   * Reads the folders on each path that the Core sent and the client builds otherwise,
   * so that the built path matches the Core's. The Core sends only the paths whose
   * folders the caller can retrieve.
   */
  async fill(payloads: Payload[]): Promise<void> {
    const keys = new Set<group.Key>();
    for (const p of payloads)
      if (p.group != null && p.path != null && this.get(p) !== p.path)
        keys.add(p.group);
    if (keys.size === 0) return;
    const groups = [...keys];
    const { store } = this.cfg.groups;
    // A cached folder on such a path is old, as on a client with no change stream.
    const cached = groups.filter((key) => store.has(key));
    const missing = groups.filter((key) => !store.has(key));
    await Promise.all([
      store.retrieve(missing),
      store.retrieve(cached, { refresh: true }),
      this.cfg.groups.retrieveAncestors(groups),
    ]);
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
    const cached = new Set<group.Key>();
    for (const { group: g } of lacking) {
      if (g == null) continue;
      if (!this.cfg.groups.store.has(g)) return onEvent("refetch");
      cached.add(g);
    }
    if (cached.size === 0) return;
    this.cfg.groups.retrieveAncestors([...cached]).catch((exc: unknown) => {
      this.cfg.onError(
        new Error("failed to read the folders of channel paths", { cause: exc }),
      );
    });
  }
}
