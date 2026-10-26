// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/components/note/Note.css";

import { Icon } from "@synnaxlabs/lyra/icon";
import { Note as Base } from "@synnaxlabs/lyra/note";
import { type ReactElement } from "react";

export type Variant = "info" | "warning" | "error";

const ICONS: Record<Variant, Icon.FC> = {
  info: Icon.Info,
  warning: Icon.Warning,
  error: Icon.Err,
};

export interface NoteProps extends Omit<Base.NoteProps, "variant"> {
  variant: Variant;
}

/** Renders a Lyra note with the icon of its variant before its content. */
export const Note = ({ variant, children, ...rest }: NoteProps): ReactElement => {
  const VariantIcon = ICONS[variant];
  return (
    <Base.Note variant={variant} {...rest}>
      <VariantIcon className="note__icon" />
      <div className="note__body">{children}</div>
    </Base.Note>
  );
};
