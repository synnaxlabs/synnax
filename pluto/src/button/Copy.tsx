// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useCallback, useState } from "react";

import { Button, type ButtonProps } from "@/button/Button";
import { Icon } from "@/icon";

const COPIED_DURATION_MS = 2000;

export interface CopyProps extends Omit<ButtonProps, "onClick"> {
  /** The text to copy to the clipboard, or a function that returns it. */
  text: string | (() => string);
  /** Optional callback invoked after successfully copying to clipboard. */
  onCopy?: () => void;
  /** Optional callback invoked if copying fails. */
  onCopyError?: (error: Error) => void;
  /** Duration in ms to show the checkmark after copying. Defaults to 2000. */
  copiedDuration?: number;
}

/**
 * A button that copies text to the clipboard and shows a checkmark on success.
 *
 * @example
 * <Button.Copy text="Hello, world!" tooltip="Copy greeting" />
 *
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
  onCopyError,
  copiedDuration = COPIED_DURATION_MS,
  tooltip = "Copy",
  children,
  ...rest
}: CopyProps): ReactElement => {
  const [copied, setCopied] = useState(false);

  const handleClick = useCallback(() => {
    void (async () => {
      try {
        const resolvedText = typeof text === "function" ? text() : text;
        await navigator.clipboard.writeText(resolvedText);
        setCopied(true);
        onCopy?.();
        setTimeout(() => setCopied(false), copiedDuration);
      } catch (err) {
        onCopyError?.(err instanceof Error ? err : new Error(String(err)));
      }
    })();
  }, [text, onCopy, onCopyError, copiedDuration]);

  return (
    <Button tooltip={copied ? "Copied!" : tooltip} onClick={handleClick} {...rest}>
      {copied ? <Icon.Check /> : <Icon.Copy />} {children}
    </Button>
  );
};
