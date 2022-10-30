import { z } from 'zod';
import { TimeSpan } from './telem';
/** Represents the connection state of a client to a synnax cluster. */
export var Connectivity;
(function (Connectivity) {
    Connectivity["Disconnected"] = "Disconnected";
    Connectivity["Connecting"] = "Connecting";
    Connectivity["Connected"] = "Connected";
    Connectivity["Failed"] = "Failed";
})(Connectivity || (Connectivity = {}));
const connectivityResponseSchema = z.object({
    clusterKey: z.string(),
});
/** Polls a synnax cluster for connectivity information. */
export default class ConnectivityClient {
    static ENDPOINT = '/connectivity/check';
    _status = Connectivity.Disconnected;
    _error;
    _statusMessage;
    pollFrequency = TimeSpan.Seconds(30);
    client;
    interval;
    onChangeHandlers;
    clusterKey;
    /**
     * @param client - The transport client to use for connectivity checks.
     * @param pollFreq - The frequency at which to poll the cluster for
     *   connectivity information.
     */
    constructor(client, pollFreq = TimeSpan.Seconds(30)) {
        this._error = undefined;
        this.client = client;
        this.pollFrequency = pollFreq;
        this.onChangeHandlers = [];
        this.check();
        this.startChecking();
    }
    /** Stops the connectivity client from polling the cluster for connectivity */
    stopChecking() {
        if (this.interval)
            clearInterval(this.interval);
    }
    /**
     * Executes a connectivity check and updates the client status and error, as
     * well as calling any registered change handlers.
     */
    async check() {
        const prev = this._status;
        try {
            const [res, err] = await this.client.send(ConnectivityClient.ENDPOINT, null, connectivityResponseSchema);
            if (!err) {
                this._status = Connectivity.Connected;
                this._statusMessage = 'Connected';
                if (res)
                    this.clusterKey = res.clusterKey;
            }
            else {
                this._status = Connectivity.Failed;
                this._error = err;
                this._statusMessage = `Connection Failed: ${this._error?.message}`;
            }
        }
        catch (err) {
            this._status = Connectivity.Failed;
            this._error = err;
            this._statusMessage = `Connection Failed: ${this._error?.message}`;
        }
        if (this.onChangeHandlers.length > 0 && prev !== this._status) {
            this.onChangeHandlers.forEach((handler) => handler(this._status, this._error, this._statusMessage));
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
    onChange(callback) {
        this.onChangeHandlers.push(callback);
    }
    startChecking() {
        this.interval = setInterval(() => {
            this.check();
        }, this.pollFrequency.milliseconds());
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiY29ubmVjdGl2aXR5LmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiLi4vLi4vLi4vc3JjL2xpYi9jb25uZWN0aXZpdHkudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQ0EsT0FBTyxFQUFFLENBQUMsRUFBRSxNQUFNLEtBQUssQ0FBQztBQUN4QixPQUFPLEVBQUUsUUFBUSxFQUFFLE1BQU0sU0FBUyxDQUFDO0FBRW5DLHVFQUF1RTtBQUN2RSxNQUFNLENBQU4sSUFBWSxZQUtYO0FBTEQsV0FBWSxZQUFZO0lBQ3RCLDZDQUE2QixDQUFBO0lBQzdCLHlDQUF5QixDQUFBO0lBQ3pCLHVDQUF1QixDQUFBO0lBQ3ZCLGlDQUFpQixDQUFBO0FBQ25CLENBQUMsRUFMVyxZQUFZLEtBQVosWUFBWSxRQUt2QjtBQUVELE1BQU0sMEJBQTBCLEdBQUcsQ0FBQyxDQUFDLE1BQU0sQ0FBQztJQUMxQyxVQUFVLEVBQUUsQ0FBQyxDQUFDLE1BQU0sRUFBRTtDQUN2QixDQUFDLENBQUM7QUFFSCwyREFBMkQ7QUFDM0QsTUFBTSxDQUFDLE9BQU8sT0FBTyxrQkFBa0I7SUFDN0IsTUFBTSxDQUFDLFFBQVEsR0FBRyxxQkFBcUIsQ0FBQztJQUN4QyxPQUFPLEdBQUcsWUFBWSxDQUFDLFlBQVksQ0FBQztJQUNwQyxNQUFNLENBQVM7SUFDZixjQUFjLENBQVU7SUFDeEIsYUFBYSxHQUFHLFFBQVEsQ0FBQyxPQUFPLENBQUMsRUFBRSxDQUFDLENBQUM7SUFDckMsTUFBTSxDQUFjO0lBQ3BCLFFBQVEsQ0FBa0I7SUFDMUIsZ0JBQWdCLENBSVg7SUFDYixVQUFVLENBQXFCO0lBRS9COzs7O09BSUc7SUFDSCxZQUFZLE1BQW1CLEVBQUUsV0FBcUIsUUFBUSxDQUFDLE9BQU8sQ0FBQyxFQUFFLENBQUM7UUFDeEUsSUFBSSxDQUFDLE1BQU0sR0FBRyxTQUFTLENBQUM7UUFDeEIsSUFBSSxDQUFDLE1BQU0sR0FBRyxNQUFNLENBQUM7UUFDckIsSUFBSSxDQUFDLGFBQWEsR0FBRyxR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