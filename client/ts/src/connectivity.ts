// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { UnaryClient } from "@synnaxlabs/freighter";
import { TimeSpan } from "@synnaxlabs/x";
import { z } from "zod";

export type ConnectionStatus = "disconnected" | "connecting" | "connected" | "failed";

export interface ConnectionState {
  status: ConnectionStatus;
  error?: Error;
  message?: string;
  clusterKey: string;
}

const connectivityResponseSchema = z.object({
  clusterKey: z.string(),
});

const DEFAULT: ConnectionState = {
  clusterKey: "",
  status: "disconnected",
  error: undefined,
  message: "Disconnected",
};

/** Polls a synnax cluster for connectivity information. */
export class Connectivity {
  private readonly id: string;
  private static readonly ENDPOINT = "/connectivity/check";
  static readonly DEFAULT: ConnectionState = DEFAULT;
  readonly state: ConnectionState;
  private readonly pollFrequency = TimeSpan.seconds(30);
  private readonly client: UnaryClient;
  private interval?: NodeJS.Timeout;
  private readonly onChangeHandlers: Array<(state: ConnectionState) => void> = [];

  /**
   * @param client - The transport client to use for connectivity checks.
   * @param pollFreq - The frequency at which to poll the cluster for
   *   connectivity information.
   */
  constructor(client: UnaryClient, pollFreq: TimeSpan = TimeSpan.seconds(30)) {
    this.state = { ...DEFAULT };
    this.id = Math.random().toString(36).substring(7);
    this.client = client;
    this.pollFrequency = pollFreq;
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
  async check(): Promise<ConnectionState> {
    const prevStatus = this.state.status;
    try {
      const [res, err] = await this.client.send(
        Connectivity.ENDPOINT,
        null,
        connectivityResponseSchema
      );
      if (err != null) throw err;
      this.state.status = "connected";
      this.state.message = "Connected";
      this.state.clusterKey = res.clusterKey;
    } catch (err) {
      this.state.status = "failed";
      this.state.error = err as Error;
      this.state.message = this.state.error.message;
    }
    if (this.onChangeHandlers.length > 0 && prevStatus !== this.state.status) {
      this.onChangeHandlers.forEach((handler) => handler(this.state));
    }
    return this.state;
  }

  /** @param callback - The function to call when the client status changes. */
  onChange(callback: (state: ConnectionState) => void): void {
    this.onChangeHandlers.push(callback);
  }

  private startChecking(): void {
    this.interval = setInterval(() => {
      void this.check();
    }, this.pollFrequency.milliseconds);
  }
}
