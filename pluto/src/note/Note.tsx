import { Align } from "@/align";
import { CSS } from "@/css";
import { Status } from "@/status";
import { PropsWithChildren } from "react";
import "@/note/Note.css";

export interface NoteProps extends Align.SpaceProps<"div"> {
  variant: Status.Variant;
}

export const Note = ({
  variant,
  className,
  children,
  ...props
}: NoteProps): JSX.Element => (
  <Align.Space
    className={CSS(className, CSS.B("note"), CSS.M(variant))}
    align="stretch"
    empty
    {...props}
  >
    {children}
  </Align.Space>
);
