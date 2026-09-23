import { type text } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { Text } from "../text";
/** Props for {@link Title}. */
export interface TitleProps extends Omit<Text.TextProps, "divided" | "level"> {
    /** Overrides the level the enclosing {@link Header} sets. */
    level?: text.Level;
}
/** The title of a {@link Header}. It takes its type scale step from the header. */
export declare const Title: ({ className, level: propsLevel, ...rest }: TitleProps) => ReactElement;
//# sourceMappingURL=Title.d.ts.map