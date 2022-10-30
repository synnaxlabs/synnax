"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.Connectivity = void 0;
const zod_1 = require("zod");
const telem_1 = require("./telem");
/** Represents the connection state of a client to a synnax cluster. */
var Connectivity;
(function (Connectivity) {
    Connectivity["Disconnected"] = "Disconnected";
    Connectivity["Connecting"] = "Connecting";
    Connectivity["Connected"] = "Connected";
    Connectivity["Failed"] = "Failed";
})(Connectivity = exports.Connectivity || (exports.Connectivity = {}));
const connectivityResponseSchema = zod_1.z.object({
    clusterKey: zod_1.z.string(),
});
/** Polls a synnax cluster for connectivity information. */
class ConnectivityClient {
    /**
     * @param client - The transport client to use for connectivity checks.
     * @param pollFreq - The frequency at which to poll the cluster for
     *   connectivity information.
     */
    constructor(client, pollFreq = telem_1.TimeSpan.Seconds(30)) {
        this._status = Connectivity.Disconnected;
        this.pollFrequency = telem_1.TimeSpan.Seconds(30);
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
        var _a, _b;
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
                this._statusMessage = `Connection Failed: ${(_a = this._error) === null || _a === void 0 ? void 0 : _a.message}`;
            }
        }
        catch (err) {
            this._status = Connectivity.Failed;
            this._error = err;
            this._statusMessage = `Connection Failed: ${(_b = this._error) === null || _b === void 0 ? void 0 : _b.message}`;
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
exports.default = ConnectivityClient;
ConnectivityClient.ENDPOINT = '/connectivity/check';
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiY29ubmVjdGl2aXR5LmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiLi4vLi4vLi4vc3JjL2xpYi9jb25uZWN0aXZpdHkudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6Ijs7O0FBQ0EsNkJBQXdCO0FBQ3hCLG1DQUFtQztBQUVuQyx1RUFBdUU7QUFDdkUsSUFBWSxZQUtYO0FBTEQsV0FBWSxZQUFZO0lBQ3RCLDZDQUE2QixDQUFBO0lBQzdCLHlDQUF5QixDQUFBO0lBQ3pCLHVDQUF1QixDQUFBO0lBQ3ZCLGlDQUFpQixDQUFBO0FBQ25CLENBQUMsRUFMVyxZQUFZLEdBQVosb0JBQVksS0FBWixvQkFBWSxRQUt2QjtBQUVELE1BQU0sMEJBQTBCLEdBQUcsT0FBQyxDQUFDLE1BQU0sQ0FBQztJQUMxQyxVQUFVLEVBQUUsT0FBQyxDQUFDLE1BQU0sRUFBRTtDQUN2QixDQUFDLENBQUM7QUFFSCwyREFBMkQ7QUFDM0QsTUFBcUIsa0JBQWtCO0lBZXJDOzs7O09BSUc7SUFDSCxZQUFZLE1BQW1CLEVBQUUsV0FBcUIsZ0JBQVEsQ0FBQyxPQUFPLENBQUMsRUFBRSxDQUFDO1FBbEJsRSxZQUFPLEdBQUcsWUFBWSxDQUFDLFlBQVksQ0FBQztRQUdwQyxrQkFBYSxHQUFHLGdCQUFRLENBQUMsT0FBTyxDQUFDLEVBQUUsQ0FBQyxDQUFDO1FBZ0IzQyxJQUFJLENBQUMsTUFBTSxHQUFHLFNBQVMsQ0FBQztRQUN4QixJQUFJLENBQUMsTUFBTSxHQUFHLE1BQU0sQ0FBQztRQUNyQixJQUFJLENBQUMsYUFBYSxHQUFHLFFBQVEsQ0FBQztRQUM5QixJQUFJLENBQUMsZ0JBQWdCLEdBQUcsRUFBRSxDQUFDO1FBQzNCLElBQUksQ0FBQyxLQUFLLEVBQUUsQ0FBQztRQUNiLElBQUksQ0FBQyxhQUFhLEVBQUUsQ0FBQztJQUN2QixDQUFDO0lBRUQsOEVBQThFO0lBQzlFLFlBQVk7UUFDVixJQUFJLElBQUksQ0FBQyxRQUFRO1lBQUUsYUFBYSxDQUFDLElBQUksQ0FBQyxRQUFRLENBQUMsQ0FBQztJQUNsRCxDQUFDO0lBRUQ7OztPQUdHO0lBQ0gsS0FBSyxDQUFDLEtBQUs7O1FBQ1QsTUFBTSxJQUFJLEdBQUcsSUFBSSxDQUFDLE9BQU8sQ0FBQztRQUMxQixJQUFJO1lBQ0YsTUFBTSxDQUFDLEdBQUcsRUFBRSxHQUFHLENBQUMsR0FBRyxNQUFNLElBQUksQ0FBQyxNQUFNLENBQUMsSUFBSSxDQUN2QyxrQkFBa0IsQ0FBQyxRQUFRLEVBQzNCLElBQUksRUFDSiwwQkFBMEIsQ0FDM0IsQ0FBQztZQUNGLElBQUksQ0FBQyxHQUFHLEVBQUU7Z0JBQ1IsSUFBSSxDQUFDLE9BQU8sR0FBRyxZQUFZLENBQUMsU0FBUyxDQUFDO2dCQUN0QyxJQUFJLENBQUMsY0FBYyxHQUFHLFdBQVcsQ0FBQztnQkFDbEMsSUFBSSxHQUFHO29CQUFFLElBQUksQ0FBQyxVQUFVLEdBQUcsR0FBRyxDQUFDLFVBQVUsQ0FBQzthQUMzQztpQkFBTTtnQkFDTCxJQUFJLENBQUMsT0FBTyxHQUFHLFlBQVksQ0FBQyxNQUFNLENBQUM7Z0JBQ25DLElBQUksQ0FBQyxNQUFNLEdBQUcsR0FBRyxDQUFDO2dCQUNsQixJQUFJLENBQUMsY0FBYyxHQUFHLHNCQUFzQixNQUFBLElBQUksQ0FBQyxNQUFNLDBDQUFFLE9BQU8sRUFBRSxDQUFDO2FBQ3BFO1NBQ0Y7UUFBQyxPQUFPLEdBQUcsRUFBRTtZQUNaLElBQUksQ0FBQyxPQUFPLEdBQUcsWUFBWSxDQUFDLE1BQU0sQ0FBQztZQUNuQyxJQUFJLENBQUMsTUFBTSxHQUFHLEdBQVksQ0FBQztZQUMzQixJQUFJLENBQUMsY0FBYyxHQUFHLHNCQUFzQixNQUFBLElBQUksQ0FBQyxNQUFNLDBDQUFFLE9BQU8sRUFBRSxDQUFDO1NBQ3BFO1FBQ0QsSUFBSSxJQUFJLENBQUMsZ0JBQWdCLENBQUMsTUFBTSxHQUFHLENBQUMsSUFBSSxJQUFJLEtBQUssSUFBSSxDQUFDLE9BQU8sRUFBRTtZQUM3RCxJQUFJLENBQUMsZ0JBQWdCLENBQUMsT0FBTyxDQUFDLENBQUMsT0FBTyxFQUFFLEVBQUUsQ0FDeEMsT0FBTyxDQUFDLElBQUksQ0FBQyxPQUFPLEVBQUUsSUFBSSxDQUFDLE1BQU0sRUFBRSxJQUFJLENBQUMsY0FBYyxDQUFDLENBQ3hELENBQUM7U0FDSDtJQUNILENBQUM7SUFFRDs7O09BR0c7SUFDSCxLQUFLO1FBQ0gsT0FBTyxJQUFJLENBQUMsTUFBTSxDQUFDO0lBQ3JCLENBQUM7SUFFRCxpREFBaUQ7SUFDakQsTUFBTTtRQUNKLE9BQU8sSUFBSSxDQUFDLE9BQU8sQ0FBQztJQUN0QixDQUFDO0lBRUQsd0VBQXdFO0lBQ3hFLGFBQWE7UUFDWCxPQUFPLElBQUksQ0FBQyxjQUFjLENBQUM7SUFDN0IsQ0FBQztJQUVELDZFQUE2RTtJQUM3RSxRQUFRLENBQ04sUUFBeUU7UUFFekUsSUFBSSxDQUFDLGdCQUFnQixDQUFDLElBQUksQ0FBQyxRQUFRLENBQUMsQ0FBQztJQUN2QyxDQUFDO0lBRU8sYUFBYTtRQUNuQixJQUFJLENBQUMsUUFBUSxHQUFHLFdBQVcsQ0FBQyxHQUFHLEVBQUU7WUFDL0IsSUFBSSxDQUFDLEtBQUssRUFBRSxDQUFDO1FBQ2YsQ0FBQyxFQUFFLElBQUksQ0FBQyxhQUFhLENBQUMsWUFBWSxFQUFFLENBQUMsQ0FBQztJQUN4QyxDQUFDOztBQWhHSCxxQ0FpR0M7QUFoR2dCLDJCQUFRLEdBQUcscUJBQXFCLENBQUMifQ==