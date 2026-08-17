// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/connection/Indicator.css";

import { Status, Synnax } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { useHeldChecking } from "@/platform/connection/useHeldChecking";
import { CSS } from "@/platform/css";

export interface IndicatorProps extends Omit<Status.IndicatorProps, "variant"> {}

/**
 * The live connection as a dot. Pulses while a check runs, keeping the variant's
 * color: an attempt in flight is activity, not a better verdict.
 */
export const Indicator = ({ className, ...rest }: IndicatorProps): ReactElement => {
  const { variant, details } = Synnax.useConnectionStatus();
  const checking = useHeldChecking(details.checking);
  return (
    <Status.Indicator
      variant={variant}
      className={CSS(
        CSS.B("connection-indicator"),
        checking && CSS.M("checking"),
        className,
      )}
      {...rest}
    />
  );
};
