// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/window/Buttons.css";

import { Button as BaseButton } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { type ReactElement, type ReactNode, useEffect, useState } from "react";

import { CSS } from "@/platform/css";

type Action = "close" | "minimize" | "maximize";

export interface ButtonsProps extends Flex.BoxProps {
  /** The buttons to leave out. */
  disabled?: Action[];
  maximized?: boolean;
  onMinimize?: () => void;
  onMaximize?: () => void;
  onClose?: () => void;
}

const useWindowFocused = (): boolean => {
  const [focused, setFocused] = useState(() => document.hasFocus());
  useEffect(() => {
    const handleFocus = () => setFocused(true);
    const handleBlur = () => setFocused(false);
    window.addEventListener("focus", handleFocus);
    window.addEventListener("blur", handleBlur);
    return () => {
      window.removeEventListener("focus", handleFocus);
      window.removeEventListener("blur", handleBlur);
    };
  }, []);
  return focused;
};

interface GlyphProps {
  children: ReactNode;
}

/** Draws the 10 px, 1 px stroke caption glyphs of Windows 11. */
const Glyph = ({ children }: GlyphProps): ReactElement => (
  <svg
    className={CSS.BE("window", "glyph")}
    viewBox="0 0 10 10"
    fill="none"
    stroke="currentColor"
    strokeWidth={1}
    aria-hidden
  >
    {children}
  </svg>
);

const MINIMIZE = (
  <Glyph>
    <path d="M0 5.5h10" />
  </Glyph>
);

const MAXIMIZE = (
  <Glyph>
    <rect x={0.5} y={0.5} width={9} height={9} rx={1.5} />
  </Glyph>
);

const RESTORE = (
  <Glyph>
    <rect x={0.5} y={2.5} width={7} height={7} rx={1.5} />
    <path d="M2.5 2.5v-.5a1.5 1.5 0 0 1 1.5-1.5h4a1.5 1.5 0 0 1 1.5 1.5v4a1.5 1.5 0 0 1-1.5 1.5h-.5" />
  </Glyph>
);

const CLOSE = (
  <Glyph>
    <path d="M0.5 0.5l9 9M9.5 0.5l-9 9" />
  </Glyph>
);

/** The minimize, maximize, and close buttons a window draws for itself on Windows. */
export const Buttons = ({
  disabled = [],
  maximized = false,
  onMinimize,
  onMaximize,
  onClose,
  className,
  ...rest
}: ButtonsProps): ReactElement => {
  const focused = useWindowFocused();
  const maximizeLabel = maximized ? "Restore" : "Maximize";
  return (
    <Flex.Box
      x
      empty
      role="group"
      aria-label="Window controls"
      className={CSS.cls(
        CSS.BE("window", "buttons"),
        !focused && CSS.M("inactive"),
        className,
      )}
      {...rest}
    >
      <Button
        action="minimize"
        aria-label="Minimize"
        tooltip="Minimize"
        onClick={onMinimize}
        disabled={disabled.includes("minimize")}
      >
        {MINIMIZE}
      </Button>
      <Button
        action="maximize"
        aria-label={maximizeLabel}
        tooltip={maximizeLabel}
        onClick={onMaximize}
        disabled={disabled.includes("maximize")}
      >
        {maximized ? RESTORE : MAXIMIZE}
      </Button>
      <Button
        action="close"
        aria-label="Close"
        tooltip="Close"
        onClick={onClose}
        disabled={disabled.includes("close")}
      >
        {CLOSE}
      </Button>
    </Flex.Box>
  );
};

interface ButtonProps extends BaseButton.ButtonProps {
  action: Action;
}

const Button = ({
  action,
  disabled = false,
  ...rest
}: ButtonProps): ReactElement | null =>
  !disabled ? (
    <BaseButton.Button
      className={CSS.cls(
        CSS.BE("window", "button"),
        CSS.BEM("window", "button", action),
      )}
      tabIndex={-1}
      variant="text"
      tooltipLocation={{ y: "bottom" }}
      {...rest}
    />
  ) : null;
