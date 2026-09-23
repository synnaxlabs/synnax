import "./Note.css";
import { type ReactElement } from "react";
import { Flex } from "../flex";
import { type Status } from "../status";
export interface NoteProps extends Flex.BoxProps<"div"> {
    variant: Status.Variant;
}
export declare const Note: ({ variant, className, ...rest }: NoteProps) => ReactElement;
//# sourceMappingURL=Note.d.ts.map