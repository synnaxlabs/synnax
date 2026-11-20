// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type rack,
  status,
  type Synnax,
  type task,
  TimeSpan,
} from "@synnaxlabs/client";
import { errors } from "@synnaxlabs/x";

import { type Listing, listingZ, TASK_TYPE } from "@/platform/file-browser/list";

const LIST_COMMAND_TYPE = "list";

const LIST_TIMEOUT = TimeSpan.seconds(10);

/** What a dialog reads from a session. Each change makes a new snapshot. */
export interface Snapshot {
  /** The last listing of each folder. */
  listings: ReadonlyMap<string, Listing>;
  /** The failure of the last list of each folder, until the folder lists again. */
  failures: ReadonlyMap<string, status.Status>;
  /** The folders that the session lists now. */
  loading: ReadonlySet<string>;
}

interface Outcome {
  listing?: Listing;
  failure?: status.Status;
}

/**
 * Lists the files of the Driver on one rack. It finds the rack's file browser task once
 * and keeps the last listing of each folder.
 */
export class Session {
  private readonly client: Synnax;
  private readonly rack: rack.Key;
  private browser: Promise<task.Task> | null = null;
  private snap: Snapshot = {
    listings: new Map(),
    failures: new Map(),
    loading: new Set(),
  };
  private readonly listeners = new Set<() => void>();
  private readonly pending = new Map<string, Promise<Listing>>();
  private readonly latest = new Map<string, number>();
  private requests = 0;
  private prefetching = false;
  private queued: string | null = null;

  constructor(client: Synnax, rack: rack.Key) {
    this.client = client;
    this.rack = rack;
  }

  /** @returns the snapshot of the session now. */
  readonly snapshot = (): Snapshot => this.snap;

  /**
   * Calls the listener after each change of the snapshot.
   * @returns a function that stops the calls.
   */
  readonly subscribe = (listener: () => void): (() => void) => {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  };

  /**
   * Lists a folder, sharing a request already in flight for it unless `fresh` is set.
   * Only the last request for a folder changes the snapshot. An empty path lists the
   * roots, the home folder, and the secrets folder.
   * @throws {Error} if the rack has no file browser, or if the Driver fails to list the
   * folder.
   */
  async list(path: string = "", fresh: boolean = false): Promise<Listing> {
    let request = fresh ? undefined : this.pending.get(path);
    if (request == null) {
      const id = ++this.requests;
      this.latest.set(path, id);
      request = this.request(path, id);
      this.pending.set(path, request);
      this.update(path, {});
    }
    return await request;
  }

  /**
   * Lists a folder in the background, unless it has a listing, a failure, or a request
   * in flight. One background list runs at a time. Of the folders asked for while it
   * runs, only the last one lists after it.
   */
  prefetch(path: string): void {
    const { listings, failures } = this.snap;
    if (listings.has(path) || failures.has(path) || this.pending.has(path)) return;
    if (this.prefetching) {
      this.queued = path;
      return;
    }
    this.prefetching = true;
    const next = (): void => {
      this.prefetching = false;
      const queued = this.queued;
      this.queued = null;
      if (queued != null) this.prefetch(queued);
    };
    // The folder lists again when it opens, and that shows the failure.
    this.list(path).then(next, next);
  }

  private async request(path: string, id: number): Promise<Listing> {
    try {
      const browser = await this.retrieveBrowser();
      const res = await browser.executeCommandSync({
        type: LIST_COMMAND_TYPE,
        timeout: LIST_TIMEOUT,
        args: { path },
      });
      if (res.variant === "error") throw status.toError(res);
      const listing = listingZ.parse(res.details?.data);
      this.settle(path, id, { listing });
      return listing;
    } catch (e) {
      this.settle(path, id, {
        failure: status.fromException(e, "Failed to list files"),
      });
      throw errors.fromUnknown(e);
    }
  }

  private settle(path: string, id: number, outcome: Outcome): void {
    if (this.latest.get(path) !== id) return;
    this.pending.delete(path);
    this.update(path, outcome);
  }

  private update(path: string, { listing, failure }: Outcome): void {
    const listings = new Map(this.snap.listings);
    const failures = new Map(this.snap.failures);
    if (listing != null) listings.set(path, listing);
    if (failure != null) failures.set(path, failure);
    else failures.delete(path);
    this.snap = { listings, failures, loading: new Set(this.pending.keys()) };
    this.listeners.forEach((listener) => listener());
  }

  private async retrieveBrowser(): Promise<task.Task> {
    this.browser ??= this.client.tasks.retrieve({
      type: TASK_TYPE,
      rack: this.rack,
      one: true,
    });
    try {
      return await this.browser;
    } catch (e) {
      // A rack can gain its file browser later, so a failed find runs again.
      this.browser = null;
      throw errors.fromUnknown(e);
    }
  }
}
