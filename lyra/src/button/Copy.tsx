// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CrudeTimeSpan, TimeSpan } from "@synnaxlabs/x";
import {
  Children,
  isValidElement,
  type MouseEvent,
  type ReactElement,
  useState,
} from "react";

import { Button, type ButtonProps } from "@/button/Button";
import { clipboard } from "@/clipboard";
import { Icon } from "@/icon";
import { type Status } from "@/status";
import { useAdder, useErrorHandler } from "@/status/core";

const COPIED_DURATION_MS = TimeSpan.seconds(2).milliseconds;

const toObject = (heading: Status.Heading): Exclude<Status.Heading, string> =>
  typeof heading === "string" ? { message: heading, name: "" } : heading;

/** Props for {@link Copy}. */
export interface CopyProps extends ButtonProps {
  /** The text to copy, or a function returning it. The function may be async. */
  text: string | (() => string | Promise<string>);
  /** Called once the text reaches the clipboard. */
  onCopy?: () => void;
  /** How long the checkmark stays up. Defaults to 2 seconds. */
  copiedDuration?: CrudeTimeSpan;
  /**
   * The status shown on success: a message, or a message and the name of what was
   * copied. A failure carries the same name. Omit for no status.
   */
  successMessage?: Status.Heading | (() => Status.Heading);
}

/**
 * A button that copies text to the clipboard and shows a checkmark on success. It
 * falls back to the copy command where the clipboard API is absent, as on http.
 *
 * @example
 * <Button.Copy text="Hello, world!" tooltip="Copy greeting" />
 * @example
 * <Button.Copy
 *   text={JSON.stringify(data)}
 *   onCopy={() => console.log("Copied!")}
 *   variant="filled"
 * />
 */
export const Copy = ({
  text,
  onCopy,
  copiedDuration = COPIED_DURATION_MS,
  successMessage,
  tooltip = "Copy",
  children,
  onClick,
  ...rest
}: CopyProps): ReactElement => {
  const [copied, setCopied] = useState(false);
  const addStatus = useAdder();
  const handleError = useErrorHandler();

  const handleClick = (e: MouseEvent<HTMLButtonElement>) => {
    onClick?.(e);
    const success =
      successMessage == null
        ? undefined
        : toObject(
            typeof successMessage === "function" ? successMessage() : successMessage,
          );
    handleError(
      async () => {
        const resolvedText = await (typeof text === "function" ? text() : text);
        await clipboard.writeText(resolvedText);
        onCopy?.();
        if (success != null) return addStatus({ variant: "success", ...success });
        setCopied(true);
        setTimeout(
          () => setCopied(false),
          TimeSpan.fromMilliseconds(copiedDuration).milliseconds,
        );
      },
      { message: "Failed to copy to clipboard", name: success?.name ?? "" },
    );
  };

  const childArray = Children.toArray(children);
  const hasCustomIcon = isValidElement(childArray[0]);
  const icon = copied ? <Icon.Check /> : hasCustomIcon ? childArray[0] : <Icon.Copy />;

  return (
    <Button tooltip={copied ? "Copied!" : tooltip} onClick={handleClick} {...rest}>
      {icon} {hasCustomIcon ? childArray.slice(1) : children}
    </Button>
  );
};
