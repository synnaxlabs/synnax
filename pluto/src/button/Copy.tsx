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
  useCallback,
  useState,
} from "react";

import { Button, type ButtonProps } from "@/button/Button";
import { Icon } from "@/icon";
import { useAdder, useErrorHandler } from "@/status/base/Aggregator";

const COPIED_DURATION_MS = TimeSpan.seconds(2).milliseconds;

/** Props for {@link Copy}. */
export interface CopyProps extends ButtonProps {
  /** The text to copy, or a function returning it. The function may be async. */
  text: string | (() => string | Promise<string>);
  /** Called once the text reaches the clipboard. */
  onCopy?: () => void;
  /** How long the checkmark stays up. Defaults to 2 seconds. */
  copiedDuration?: CrudeTimeSpan;
  /** The status shown on success. Omit for no status. */
  successMessage?: string | (() => string);
}

/**
 * A button that copies text to the clipboard and shows a checkmark on success.
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

  const handleClick = useCallback(
    (e: MouseEvent<HTMLButtonElement>) => {
      onClick?.(e);
      handleError(async () => {
        const resolvedText = await (typeof text === "function" ? text() : text);
        await navigator.clipboard.writeText(resolvedText);
        onCopy?.();
        if (successMessage != null) {
          const message =
            typeof successMessage === "function" ? successMessage() : successMessage;
          return addStatus({ variant: "success", message });
        }
        setCopied(true);
        setTimeout(
          () => setCopied(false),
          TimeSpan.fromMilliseconds(copiedDuration).milliseconds,
        );
      });
    },
    [text, onCopy, copiedDuration, addStatus, successMessage, handleError, onClick],
  );

  const childArray = Children.toArray(children);
  const hasCustomIcon = isValidElement(childArray[0]);
  const icon = copied ? <Icon.Check /> : hasCustomIcon ? childArray[0] : <Icon.Copy />;

  return (
    <Button tooltip={copied ? "Copied!" : tooltip} onClick={handleClick} {...rest}>
      {icon} {hasCustomIcon ? childArray.slice(1) : children}
    </Button>
  );
};
