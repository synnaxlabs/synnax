import { Align } from "@/align";
import { CSS } from "@/css";
import { Status } from "@/status";
import { PropsWithChildren } from "react";
import "@/note/Note.css";

export interface NoteProps extends PropsWithChildren<{}> {
  variant: Status.Variant;
}

export const Note = ({ variant, children }: NoteProps): JSX.Element => (
  <Align.Space className={CSS(CSS.B("note"), CSS.M(variant))} align="stretch" empty>
    {children}
  </Align.Space>
);
