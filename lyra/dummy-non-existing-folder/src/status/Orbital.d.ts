import "./Orbital.css";
import { type ReactElement, type ReactNode } from "react";
import { Flex } from "../flex";
export interface OrbitalProps extends Flex.BoxProps {
    /**
     * Occupies the orbital's center in place of the default sphere. Give it an
     * opaque background so ring arcs passing behind are occluded.
     */
    core?: ReactNode;
}
/**
 * Orbital is the large loading figure: two laser rings tumbling on crossed axes
 * plus a slow dashed ring on a third, around a central core. Sized by
 * `--pluto-orbital-size`. It carries no appearance delay of its own; wrap it in
 * {@link Loading} for a surface that may resolve quickly.
 */
export declare const Orbital: ({ core, className, ...rest }: OrbitalProps) => ReactElement;
//# sourceMappingURL=Orbital.d.ts.map