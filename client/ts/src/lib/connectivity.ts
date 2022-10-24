import { UnaryClient } from '@synnaxlabs/freighter';
import { z } from 'zod';
import { TimeSpan } from './telem';

/** Represents the connection state of a client to a synnax cluster. */
export enum Connectivity {
  DISCNNECTED = 'DISCONNECTED',
  CONNECTING = 'CONNECTING',
  CONNECTED = 'CONNECTED',
  FAILED = 'FAILED',
}

const connectivityResponseSchema = z.object({
  clusterKey: z.string(),
});

/** Polls a synnax cluster for connectivity information. */
export default class ConnectivityClient {
  private static ENDPOINT = '/connectivity/check';
  private _status = Connectivity.DISCNNECTED;
  private _error?: Error;
  private _statusMessage?: string;
  private pollFrequency = TimeSpan.Seconds(30);
  private client: UnaryClient;
  private interval?: NodeJS.Timeout;
  private onChangeHandlers: ((
    status: Connectivity,
    error?: Error,
    message?: string
  ) => void)[];
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
    this.startChecking();
  }

  /** Stops the connectivity client from polling the cluster for connectivity */
  stopChecking() {
    if (this.interval) clearInterval(this.interval);
  }

  /**
   * Executes a connectivity check and updates the client status and error, as
   * well as calling any registered change handlers.
   */
  async check() {
    const prev = this._status;
    try {
      const [res, err] = await this.client.send(
        ConnectivityClient.ENDPOINT,
        null,
        connectivityResponseSchema
      );
      if (!err) {
        this._status = Connectivity.CONNECTED;
        this._statusMessage = 'Connected';
        if (res) this.clusterKey = res.clusterKey;
      } else {
        this._status = Connectivity.FAILED;
        this._error = err;
        this._statusMessage = `Connection Failed: ${this._error?.message}`;
      }
    } catch (err) {
      this._status = Connectivity.FAILED;
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
  error() {
    return this._error;
  }

  /** @returns The current status of the client. */
  status() {
    return this._status;
  }

  /** @returns A status message describing the current connection state */
  statusMessage() {
    return this._statusMessage;
  }

  /** @param callback - The function to call when the client status changes. */
  onChange(
    callback: (status: Connectivity, error?: Error, message?: string) => void
  ) {
    this.onChangeHandlers.push(callback);
  }

  private startChecking() {
    this.interval = setInterval(() => {
      this.check();
    }, this.pollFrequency.milliseconds());
  }
}
