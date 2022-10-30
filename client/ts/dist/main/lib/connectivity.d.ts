import { UnaryClient } from '@synnaxlabs/freighter';
import { TimeSpan } from './telem';
/** Represents the connection state of a client to a synnax cluster. */
export declare enum Connectivity {
    Disconnected = "Disconnected",
    Connecting = "Connecting",
    Connected = "Connected",
    Failed = "Failed"
}
/** Polls a synnax cluster for connectivity information. */
export default class ConnectivityClient {
    private static ENDPOINT;
    private _status;
    private _error?;
    private _statusMessage?;
    private pollFrequency;
    private client;
    private interval?;
    private onChangeHandlers;
    clusterKey: string | undefined;
    /**
     * @param client - The transport client to use for connectivity checks.
     * @param pollFreq - The frequency at which to poll the cluster for
     *   connectivity information.
     */
    constructor(client: UnaryClient, pollFreq?: TimeSpan);
    /** Stops the connectivity client from polling the cluster for connectivity */
    stopChecking(): void;
    /**
     * Executes a connectivity check and updates the client status and error, as
     * well as calling any registered change handlers.
     */
    check(): Promise<void>;
    /**
     * @returns The error that caused the last connectivity check to fail.
     *   Undefined if the last check was successful.
     */
    error(): Error | undefined;
    /** @returns The current status of the client. */
    status(): Connectivity;
    /** @returns A status message describing the current connection state */
    statusMessage(): string | undefined;
    /** @param callback - The function to call when the client status changes. */
    onChange(callback: (status: Connectivity, error?: Error, message?: string) => void): void;
    private startChecking;
}
