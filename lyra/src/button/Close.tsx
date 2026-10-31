// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Button, type ButtonProps, type ElementType } from "@/button/Button";
import { CSS } from "@/css";
import { Icon } from "@/icon";

/** Props for {@link Close}. */
export interface CloseProps extends Omit<ButtonProps, "el"> {
  /** Renders a `div` inside a host that is itself a button. */
  el?: Extract<ElementType, "button" | "div">;
}

const CLOSE_ICON = <Icon.Close />;

/**
 * A destructive glyph close that overlays its host's leading icon and reveals on
 * host hover. The host marks itself as a reveal container. Not a tab stop: the
 * keyboard path to closing belongs to the host (e.g. Delete on a focused tab).
 */
export const Close = ({
  className,
  children = CLOSE_ICON,
  ...rest
}: CloseProps): ReactElement => (
  <Button<ElementType>
    aria-label="Close"
    variant="glyph"
    sharp
    tabIndex={-1}
    reveal
    className={CSS.cls(CSS.BM("btn", "close"), CSS.M("destructive"), className)}
    {...rest}
  >
    {children}
  </Button>
);
