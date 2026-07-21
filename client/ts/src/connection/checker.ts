// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import {
  ClockSkewCalculator,
  type CrudeTimeSpan,
  errors,
  migrate,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { z } from "zod";

export const statusZ = z.enum(["disconnected", "connecting", "connected", "failed"]);
export type Status = z.infer<typeof statusZ>;

export const stateZ = z.object({
  status: statusZ,
  error: z.instanceof(Error).optional(),
  message: z.string().optional(),
  clusterKey: z.string(),
  clientVersion: z.string(),
  clientServerCompatible: z.boolean(),
  nodeVersion: z.string().optional(),
  clockSkew: TimeSpan.z.default(TimeSpan.ZERO),
  clockSkewExceeded: z.boolean().default(false),
});
export interface State extends z.infer<typeof stateZ> {}

const responseZ = z.object({
  clusterKey: z.string(),
  nodeVersion: z.string().optional(),
  nodeTime: TimeStamp.z,
});
const requestZ = z.void();

const DEFAULT: State = {
  clusterKey: "",
  status: "disconnected",
  error: undefined,
  message: "Disconnected",
  clientServerCompatible: false,
  clientVersion: __VERSION__,
  clockSkew: TimeSpan.ZERO,
  clockSkewExceeded: false,
};

const createWarning = (
  nodeVersion: string | null,
  clientVersion: string,
  clientIsNewer: boolean,
): string => {
  const toUpgrade = clientIsNewer ? "Core" : "client";
  return `The Synnax Core version ${nodeVersion != null ? `${nodeVersion} ` : ""}is too ${clientIsNewer ? "old" : "new"} for client version ${clientVersion}.
  This may cause compatibility issues. We recommend updating the ${toUpgrade}. For more information, see
  https://docs.synnaxlabs.com/reference/client/resources/troubleshooting#old-${toUpgrade.toLowerCase()}-version`;
};

/** Polls a synnax cluster for connectivity information. */
export class Checker {
  static readonly DEFAULT: State = DEFAULT;
  private readonly liveState: State;
  private readonly pollFrequency: TimeSpan;
  private readonly client: UnaryClient;
  private readonly name?: string;
  private interval?: NodeJS.Timeout;
  private readonly clientVersion: string;
  private readonly onChangeHandlers: Array<(state: State) => void> = [];
  static readonly connectionStateZ = stateZ;
  private versionWarned = false;
  private readonly skewCalc: ClockSkewCalculator;
  private readonly clockSkewThreshold: TimeSpan;
  private checking = false;

  /**
   * @param client - The transport client to use for connectivity checks.
   * @param pollFreq - The frequency at which to poll the cluster for
   *   connectivity information.
   */
  constructor(
    client: UnaryClient,
    pollFreq: CrudeTimeSpan = TimeSpan.seconds(30),
    clientVersion: string,
    name?: string,
    clockSkewThreshold: CrudeTimeSpan = TimeSpan.seconds(1),
  ) {
    this.liveState = { ...DEFAULT };
    this.client = client;
    this.pollFrequency = new TimeSpan(pollFreq);
    this.clientVersion = clientVersion;
    this.name = name;
    this.skewCalc = new ClockSkewCalculator();
    this.clockSkewThreshold = new TimeSpan(clockSkewThreshold).abs();
    void this.check();
    this.start();
  }

  /** Stops the connectivity client from polling the cluster for connectivity */
  stop(): void {
    if (this.interval != null) clearInterval(this.interval);
  }

  /**
   * Executes a connectivity check and updates the client status and error, as
   * well as calling any registered change handlers.
   */
  async check(): Promise<State> {
    const prevStatus = this.liveState.status;
    const prevSkewExceeded = this.liveState.clockSkewExceeded;
    const measureSkew = !this.checking;
    this.checking = true;
    try {
      if (measureSkew) this.skewCalc.start();
      const res = await this.client.send(
        "/connectivity/check",
        undefined,
        requestZ,
        responseZ,
      );
      if (measureSkew) {
        this.skewCalc.end(res.nodeTime);
        this.liveState.clockSkew = this.skewCalc.skew;
        this.liveState.clockSkewExceeded = this.skewCalc.exceeds(
          this.clockSkewThreshold,
        );
        if (this.liveState.clockSkewExceeded) {
          const direction = this.skewCalc.skew.valueOf() > 0n ? "ahead of" : "behind";
          console.warn(
            `Measured excessive clock skew between this host and ` +
              `the Synnax Core. This host is ${direction} the Synnax Core ` +
              `by approximately ${this.skewCalc.skew.abs().toString()}.`,
          );
        }
      }
      const nodeVersion = res.nodeVersion;
      const { clientVersion } = this;
      if (nodeVersion == null) {
        this.liveState.clientServerCompatible = false;
        if (!this.versionWarned) {
          console.warn(createWarning(null, clientVersion, true));
          this.versionWarned = true;
        }
      } else if (
        !migrate.versionsEqual(clientVersion, nodeVersion, {
          checkMajor: true,
          checkMinor: true,
          checkPatch: false,
        })
      ) {
        this.liveState.clientServerCompatible = false;
        if (!this.versionWarned) {
          console.warn(
            createWarning(
              nodeVersion,
              clientVersion,
              migrate.semVerNewer(clientVersion, nodeVersion),
            ),
          );
          this.versionWarned = true;
        }
      } else this.liveState.clientServerCompatible = true;
      this.liveState.status = "connected";
      this.liveState.message = `Connected to ${this.name ?? "cluster"}`;
      this.liveState.clusterKey = res.clusterKey;
      this.liveState.nodeVersion = res.nodeVersion;
      this.liveState.clientVersion = this.clientVersion;
    } catch (err) {
      this.liveState.status = "failed";
      this.liveState.error = errors.fromUnknown(err);
      this.liveState.message = this.state.error?.message;
    } finally {
      this.checking = false;
    }
    const changed =
      prevStatus !== this.liveState.status ||
      prevSkewExceeded !== this.liveState.clockSkewExceeded;
    if (this.onChangeHandlers.length > 0 && changed)
      this.onChangeHandlers.forEach((handler) => handler(this.state));
    return this.state;
  }

  /** @returns a copy of the current client state. */
  get state(): State {
    return { ...this.liveState };
  }

  /** @param callback - The function to call when the client status changes. */
  onChange(callback: (state: State) => void): void {
    this.onChangeHandlers.push(callback);
  }

  private start(): void {
    this.interval = setInterval(() => {
      void this.check();
    }, this.pollFrequency.milliseconds);
  }
}
