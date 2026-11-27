// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Key, Payload, RetrieveRequest, Where } from "@/channel/types.gen";
import type { group } from "@/group";
import { query } from "@/query";

/** What a path reads of a channel. */
interface Placed {
  key: Key;
  name: string;
  group?: group.Key;
}

/** Joins the names on a path. Matches the separator of the Core. */
const SEPARATOR = ".";

/** @returns whether the where, or a where inside it, filters by path. */
const filtersPath = (where?: Where): boolean =>
  where != null &&
  (where.path != null ||
    where.and?.some(filtersPath) === true ||
    where.or?.some(filtersPath) === true ||
    filtersPath(where.not));

/** @returns whether the event can change the path of a channel folder. */
const movesFolder = (event: query.TableEvent<group.Key, group.Group>): boolean =>
  event.variant === "delete" ||
  (event.value.scope === "channel" && query.isChange(event));

export interface PathsConfig<C extends Placed> {
  /** The table of the channels. */
  channels: query.Table<Key, C>;
  /** The groups, whose table holds the folders that the paths pass. */
  groups: group.Client;
  /** Sends a retrieve to the Core. */
  retrieve: (request: RetrieveRequest) => Promise<Payload[]>;
}

/**
 * Builds the path of a channel from its name and the folders that the group table
 * holds, as the Core builds it. The change stream keeps both tables current, so a
 * rename or a move of a channel or a folder changes the path with no fetch. A retrieve
 * that asks for paths fetches the folders that the table lacks.
 */
export class Paths<C extends Placed> {
  /**
   * Composes again the answers that ask for paths when a folder changes, and refetches
   * them when a channel moves into a folder whose path the client lacks. Also refetches
   * an answer whose where filters by path when a channel folder changes.
   */
  readonly triggers: Array<query.Trigger<RetrieveRequest, Key>>;
  private readonly cfg: PathsConfig<C>;

  constructor(cfg: PathsConfig<C>) {
    this.cfg = cfg;
    const { channels, groups } = cfg;
    const composed: query.Trigger<RetrieveRequest, Key> = {
      attach: (params, onEvent, members) => {
        if (params.includePath !== true) return () => {};
        const offGroups = groups.store.subscribeBatch(() => {
          const keys = members();
          if (keys.length > 0) onEvent([...keys]);
        });
        const offChannels = channels.subscribeBatch((events) => {
          const lacks = events.some(
            (e) =>
              e.variant === "set" && query.isChange(e) && this.get(e.value) == null,
          );
          if (lacks) onEvent("refetch");
        });
        return () => {
          offGroups();
          offChannels();
        };
      },
      version: () => groups.store.version(),
      contentOnly: true,
    };
    const filtered = query.trigger<RetrieveRequest, Key, group.Key, group.Group>(
      groups.store,
      (event, params: RetrieveRequest) =>
        filtersPath(params.where) && movesFolder(event) ? "refetch" : null,
    );
    this.triggers = [composed, filtered];
  }

  /**
   * @returns the path of the channel, or undefined when the group table lacks a folder
   * on it.
   */
  get({ name, group: start }: Placed): string | undefined {
    const names = [name];
    const passed = new Set<group.Key>();
    // An optimistic move can make a cycle until the Core refuses it.
    for (let key = start; key != null;) {
      const folder = this.cfg.groups.store.get(key);
      if (folder == null || passed.has(key)) return undefined;
      passed.add(key);
      names.push(folder.name);
      key = folder.parent?.type === "group" ? folder.parent.key : undefined;
    }
    return names.reverse().join(SEPARATOR);
  }

  /**
   * Fetches the channels that the request selects. A request that asks for paths also
   * fetches the folders that the group table lacks on each path that the Core shows,
   * since the Core shows only the paths whose folders the caller can retrieve.
   */
  async fetch(request: RetrieveRequest): Promise<Payload[]> {
    const payloads = await this.cfg.retrieve(request);
    if (request.includePath !== true) return payloads;
    const lacking = new Set<group.Key>();
    for (const payload of payloads)
      if (payload.group != null && payload.path != null && this.get(payload) == null)
        lacking.add(payload.group);
    if (lacking.size === 0) return payloads;
    const keys = [...lacking];
    await Promise.all([
      this.cfg.groups.retrieve({ where: { key: { in: keys } } }),
      this.cfg.groups.retrieveAncestors(keys),
    ]);
    return payloads;
  }
}
