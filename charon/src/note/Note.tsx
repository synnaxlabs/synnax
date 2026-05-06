// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/note/Note.css";

import { type status } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Flex } from "@/flex";

export interface NoteProps extends Flex.BoxProps<"div"> {
  variant: status.Variant;
}

export const Note = ({ variant, className, ...rest }: NoteProps): ReactElement => (
  <Flex.Box
    className={CSS(className, CSS.B("note"), CSS.M(variant))}
    align="stretch"
    empty
    {...rest}
  />
);
