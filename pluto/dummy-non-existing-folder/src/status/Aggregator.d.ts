import { Status as Base } from "@synnaxlabs/lyra/status";
import { type ReactElement } from "react";
/** Props for {@link Aggregator}. */
export interface AggregatorProps extends Base.AggregatorProps {
}
/**
 * The lyra aggregator plus a bridge that forwards statuses reported on the aether
 * worker thread. Mount one near the root of the app, inside the Aether provider.
 */
export declare const Aggregator: ({ children, ...rest }: AggregatorProps) => ReactElement;
//# sourceMappingURL=Aggregator.d.ts.map