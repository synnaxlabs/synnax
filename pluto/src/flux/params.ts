import { type primitive } from "@synnaxlabs/x";

/**
 * Parameters used to retrieve and or/update a resource from within a query. The query
 * re-executes whenever the parameters change.
 */
export type Params = Record<string, primitive.Value>;
