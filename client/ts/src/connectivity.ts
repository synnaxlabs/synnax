// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { UnaryClient } from "@synnaxlabs/freighter";
import { z } from "zod";

import { TimeSpan } from "./telem";

export type Connectivity = "disconnected" | "connecting" | "connected" | "failed";

const connectivityResponseSchema = z.object({
  clusterKey: z.string(),
});

/** Polls a synnax cluster for connectivity information. */
export default class ConnectivityClient {
  private static readonly ENDPOINT = "/connectivity/check";
  private _status: Connectivity = "disconnected";
  private _error?: Error;
  private _statusMessage?: string;
  private readonly pollFrequency = TimeSpan.Seconds(30);
  private readonly client: UnaryClient;
  private interval?: NodeJS.Timeout;
  private readonly onChangeHandlers: Array<
    (status: Connectivity, error?: Error, message?: string) => void
  >;

  clusterKey: string | undefined;

  /**
   * @param client - The transport client to use for connectivity checks.
   * @param pollFreq - The frequency at which to poll the cluster for
   *   connectivity information.
   */
  constructor(client: UnaryClient, pollFreq: TimeSpan = TimeSpan.Seconds(30)) {
    this._error = undefined;
    this.client = client;
    this.pollFrequency = pollFreq;
    this.onChangeHandlers = [];
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
  async check(): Promise<void> {
    const prev = this._status;
    try {
      const [res, err] = await this.client.send(
        ConnectivityClient.ENDPOINT,
        null,
        connectivityResponseSchema
      );
      if (err != null) throw err;
      this._status = "connected";
      this._statusMessage = "Connected";
      if (res != null) this.clusterKey = res.clusterKey;
    } catch (err) {
      this._status = "failed";
      this._error = err as Error;
      this._statusMessage = `Connection Failed: ${this._error?.message}`;
    }
    if (this.onChangeHandlers.length > 0 && prev !== this._status) {
      this.onChangeHandlers.forEach((handler) =>
        handler(this._status, this._error, this._statusMessage)
      );
    }
  }

  /**
   * @returns The error that caused the last connectivity check to fail.
   *   Undefined if the last check was successful.
   */
  error(): Error | undefined {
    return this._error;
  }

  /** @returns The current status of the client. */
  status(): Connectivity {
    return this._status;
  }

  /** @returns A status message describing the current connection state */
  statusMessage(): string | undefined {
    return this._statusMessage;
  }

  /** @param callback - The function to call when the client status changes. */
  onChange(
    callback: (status: Connectivity, error?: Error, message?: string) => void
  ): void {
    this.onChangeHandlers.push(callback);
  }

  private startChecking(): void {
    this.interval = setInterval(() => {
      void this.check();
    }, this.pollFrequency.milliseconds());
  }
}
