// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/input/time/Time.css";

import { state, type text } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode, useCallback } from "react";

import { type Component } from "@/component";
import { CSS } from "@/css";
import { Dialog } from "@/dialog";
import { Flex } from "@/flex";
import { type Variant } from "@/input/types";

export interface CellProps {
  /** The value as content, shown at rest. */
  label: ReactNode;
  open: boolean;
  /** Called when the trigger or a click outside asks to open or close the editor. */
  onOpenChange: (open: boolean) => void;
  variant?: Variant;
  level?: text.Level;
  size?: Component.Size;
  disabled?: boolean;
  /** Renders the label alone, with no editor behind it. */
  preview?: boolean;
  /** The exact value, shown on hover when the label rounds it. */
  tooltip?: string;
  className?: string;
  /** The text field at the top of the editor. */
  field: ReactNode;
  /** The body under the field: a preview and actions. */
  children: ReactNode;
}

/**
 * A value that reads as content and edits in a connected dialog under it, the shape
 * of a select: the trigger never changes while the editor is open.
 */
export const Cell = ({
  label,
  open,
  onOpenChange,
  variant = "outlined",
  level,
  size,
  disabled,
  preview,
  tooltip,
  className,
  field,
  children,
}: CellProps): ReactElement => {
  const handleVisibleChange = useCallback(
    (next: state.SetArg<boolean>) => onOpenChange(state.executeSetter(next, open)),
    [onOpenChange, open],
  );
  return (
    <Dialog.Frame
      variant="connected"
      visible={open}
      onVisibleChange={handleVisibleChange}
      className={CSS.cls(CSS.B("time-cell"), CSS.M(variant), className)}
    >
      <Dialog.Trigger
        hideCaret
        variant={variant === "outlined" ? "outlined" : "text"}
        level={level}
        size={size}
        disabled={disabled}
        preview={preview}
        tooltip={open ? undefined : tooltip}
        tooltipLocation="bottom"
        className={CSS.BE("time-cell", "trigger")}
      >
        {label}
      </Dialog.Trigger>
      <Dialog.Dialog className={CSS.BE("time-cell", "editor")} bordered={false}>
        {field}
        <Flex.Box y empty bordered borderColor={6} rounded full="x">
          {children}
        </Flex.Box>
      </Dialog.Dialog>
    </Dialog.Frame>
  );
};
