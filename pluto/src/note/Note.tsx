// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/note/Note.css";

import { Align } from "@/align";
import { CSS } from "@/css";
import { type Status } from "@/status";

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
