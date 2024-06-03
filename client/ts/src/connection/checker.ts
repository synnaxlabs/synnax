// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { UnaryClient } from "@synnaxlabs/freighter";
import { TimeSpan } from "@synnaxlabs/x/telem";
import { z } from "zod";

const STATUSES = ["disconnected", "connecting", "connected", "failed"] as const;
export const status = z.enum(STATUSES);
export type Status = z.infer<typeof status>;

export const state = z.object({
  status,
  error: z.instanceof(Error).optional(),
  message: z.string().optional(),
  clusterKey: z.string(),
});

export type State = z.infer<typeof state>;

const responseZ = z.object({
  clusterKey: z.string(),
});

const DEFAULT: State = {
  clusterKey: "",
  status: "disconnected",
  error: undefined,
  message: "Disconnected",
};

/** Polls a synnax cluster for connectivity information. */
export class Checker {
  private static readonly ENDPOINT = "/connectivity/check";
  static readonly DEFAULT: State = DEFAULT;
  private readonly _state: State;
  private readonly pollFrequency = TimeSpan.seconds(30);
  private readonly client: UnaryClient;
  private readonly name?: string;
  private interval?: NodeJS.Timeout;
  private readonly onChangeHandlers: Array<(state: State) => void> = [];
  static readonly connectionStateZ = state;

  /**
   * @param client - The transport client to use for connectivity checks.
   * @param pollFreq - The frequency at which to poll the cluster for
   *   connectivity information.
   */
  constructor(
    client: UnaryClient,
    pollFreq: TimeSpan = TimeSpan.seconds(30),
    name?: string,
  ) {
    this._state = { ...DEFAULT };
    this.client = client;
    this.pollFrequency = pollFreq;
    this.name = name;
    void this.check();
    this.startChecking();
  }

  /** Stops the connectivity client from polling the cluster for connectivity */
  stopChecking(): void {
    if (this.interval != null) clearInterval(this.interval);
  }

  /**
   * Executes a connectivity check and updates the client status and error, as
   * well as calling any registered change handlers.
   */
  async check(): Promise<State> {
    const prevStatus = this._state.status;
    try {
      const [res, err] = await this.client.send(Checker.ENDPOINT, {}, z.object({}), responseZ);
      if (err != null) throw err;
      this._state.status = "connected";
      this._state.message = `Connected to ${this.name ?? "cluster"}`;
      this._state.clusterKey = res.clusterKey;
    } catch (err) {
      this._state.status = "failed";
      this._state.error = err as Error;
      this._state.message = this.state.error?.message;
    }
    if (this.onChangeHandlers.length > 0 && prevStatus !== this._state.status)
      this.onChangeHandlers.forEach((handler) => handler(this.state));
    return this.state;
  }

  /** @returns a copy of the current client state. */
  get state(): State {
    return { ...this._state };
  }

  /** @param callback - The function to call when the client status changes. */
  onChange(callback: (state: State) => void): void {
    this.onChangeHandlers.push(callback);
  }

  private startChecking(): void {
    this.interval = setInterval(() => {
      void this.check();
    }, this.pollFrequency.milliseconds);
  }
}
