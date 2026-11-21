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

/**
 * Lists the files of the Driver on one rack. It finds the rack's file browser task once
 * and keeps the last listing of each folder.
 */
export class Session {
  private readonly client: Synnax;
  private readonly rack: rack.Key;
  private browser: Promise<task.Task> | null = null;
  private readonly listings = new Map<string, Listing>();
  private readonly pending = new Map<string, Promise<Listing>>();

  constructor(client: Synnax, rack: rack.Key) {
    this.client = client;
    this.rack = rack;
  }

  /** @returns the last listing of the folder, or undefined if it was never listed. */
  cached(path: string): Listing | undefined {
    return this.listings.get(path);
  }

  /**
   * Lists a folder, sharing a request already in flight for it. An empty path lists the
   * roots, the home folder, and the secrets folder.
   * @throws {Error} if the rack has no file browser, or if the Driver fails to list the
   * folder.
   */
  async list(path: string = ""): Promise<Listing> {
    let request = this.pending.get(path);
    if (request == null) {
      request = this.request(path).finally(() => this.pending.delete(path));
      this.pending.set(path, request);
    }
    return await request;
  }

  private async request(path: string): Promise<Listing> {
    const browser = await this.retrieveBrowser();
    const res = await browser.executeCommandSync({
      type: LIST_COMMAND_TYPE,
      timeout: LIST_TIMEOUT,
      args: { path },
    });
    if (res.variant === "error") throw status.toError(res);
    const listing = listingZ.parse(res.details?.data);
    this.listings.set(path, listing);
    return listing;
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
