import { connection } from "@synnaxlabs/client";
import { type CrudeTimeSpan } from "@synnaxlabs/x";
/**
 * Continuously checks the given cluster address on the interval while mounted.
 * A params change (by deep equality) restarts the loop; between checks the
 * settled status stays in place.
 * @returns null while the first check is pending or params are absent.
 */
export declare const useCheckConnection: (params?: connection.CheckParams | null, interval?: CrudeTimeSpan) => connection.Status | null;
//# sourceMappingURL=useCheckConnection.d.ts.map