import "./Animated.css";
import { type location } from "@synnaxlabs/x";
import { Icon } from "../icon";
/** Props for {@link Animated}. */
export interface AnimatedProps extends Icon.IconProps {
    /** Where the caret points while enabled. */
    enabledLoc: location.Location;
    /** Where it points otherwise. */
    disabledLoc: location.Location;
    enabled: boolean;
}
/** A caret that turns between two directions. Use it on anything that opens. */
export declare const Animated: ({ className, enabledLoc, disabledLoc, enabled, ...rest }: AnimatedProps) => import("react").JSX.Element;
//# sourceMappingURL=Animated.d.ts.map