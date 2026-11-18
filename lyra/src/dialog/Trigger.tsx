// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/dialog/Trigger.css";

import { type ReactElement } from "react";

import { Button } from "@/button";
import { Caret } from "@/caret";
import { CSS } from "@/css";
import { useContext } from "@/dialog/Frame";
import { useCombinedRefs } from "@/hooks";
import { type Icon } from "@/icon";

/** Props for {@link Trigger}. */
export interface TriggerProps extends Button.ButtonProps {
  hideCaret?: boolean;
}

/**
 * The button that opens and closes the enclosing {@link Frame}, carrying a caret that
 * turns with the dialog.
 */
export const Trigger = ({
  ref,
  onClick,
  className,
  hideCaret = false,
  children,
  preview,
  ...rest
}: TriggerProps): ReactElement => {
  const { toggle, visible, variant, triggerRef } = useContext();
  const combinedRef = useCombinedRefs(ref, triggerRef);
  let endIcon: Icon.ReactElement | undefined;
  const opensDialog = preview !== true;
  if (preview === true) hideCaret = true;
  if (variant !== "modal" && !hideCaret)
    endIcon = (
      <Caret.Animated enabled={visible} enabledLoc="bottom" disabledLoc="left" />
    );
  return (
    <Button.Button
      ref={combinedRef}
      className={CSS.cls(CSS.BE("dialog", "trigger"), className)}
      onClick={(e) => {
        onClick?.(e);
        toggle();
      }}
      full="x"
      preview={preview}
      aria-haspopup={opensDialog ? "dialog" : undefined}
      aria-expanded={opensDialog ? visible : undefined}
      {...rest}
    >
      {children}
      {endIcon}
    </Button.Button>
  );
};
