import { UnaryClient } from '@synnaxlabs/freighter';
import { TimeSpan } from './telem';

/** Represents the connection state of a client to a synnax cluster. */
export enum Connectivity {
  DISCNNECTED = 'DISCONNECTED',
  CONNECTING = 'CONNECTING',
  CONNECTED = 'CONNECTED',
  FAILED = 'FAILED',
}

/** Polls a synnax cluster for connectivity information. */
export default class ConnectivityClient {
  private static ENDPOINT = '/connectivity/check';
  private _status = Connectivity.DISCNNECTED;
  private pollFrequency = TimeSpan.Seconds(30);
  private client: UnaryClient;
  private _error?: Error;
  private interval?: NodeJS.Timeout;
  private onChangeHandlers: ((status: Connectivity, error?: Error) => void)[];

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
    const [_, err] = await this.client.send(
      ConnectivityClient.ENDPOINT,
      null,
      null
    );
    const prev = this._status;
    if (!err) {
      this._status = Connectivity.CONNECTED;
    } else {
      this._status = Connectivity.FAILED;
      this._error = err;
    }
    if (this.onChangeHandlers.length > 0 && prev !== this._status) {
      this.onChangeHandlers.forEach((handler) => handler(this._status, err));
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

  /** @param callback - The function to call when the client status changes. */
  onChange(callback: (status: Connectivity) => void) {
    this.onChangeHandlers.push(callback);
  }

  private startChecking() {
    this.interval = setInterval(this.check, this.pollFrequency.milliseconds());
  }
}
