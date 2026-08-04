// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Button, type ButtonProps } from "@/button/Button";
import { CSS } from "@/css";
import { Icon } from "@/icon";

export interface CloseProps extends ButtonProps {}

/**
 * A destructive glyph close that overlays its host's leading icon and reveals on
 * host hover. The host marks itself as a reveal container. Not a tab stop: the
 * keyboard path to closing belongs to the host (e.g. Delete on a focused tab).
 */
export const Close = ({
  className,
  children = <Icon.Close />,
  ...rest
}: CloseProps): ReactElement => (
  <Button
    aria-label="Close"
    variant="text"
    sharp
    tabIndex={-1}
    reveal
    className={CSS(
      CSS.BM("btn", "close"),
      CSS.BM("btn", "glyph"),
      CSS.M("destructive"),
      className,
    )}
    {...rest}
  >
    {children}
  </Button>
);
