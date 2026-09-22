// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Synnax } from "@synnaxlabs/pluto";
import { type PropsWithChildren, type ReactNode } from "react";

import { Activate, type ActivateProps } from "@/feature/license/Activate";

export interface GuardProps extends PropsWithChildren, ActivateProps {}

/**
 * Renders the activation screen instead of its children while the active Core refuses
 * requests for want of a license. The connection check keeps polling, so the screen
 * dismisses on its own once a license applies.
 */
export const Guard = ({ children, standalone }: GuardProps): ReactNode => {
  const status = Synnax.useConnectionStatus();
  if (status.variant === "error" && status.details.reason === "unlicensed")
    return <Activate standalone={standalone} />;
  return children;
};
