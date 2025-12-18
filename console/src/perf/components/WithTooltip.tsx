// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Tooltip } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

interface WithTooltipProps {
  tooltip?: string;
  children: ReactElement;
}

export const WithTooltip = ({ tooltip, children }: WithTooltipProps): ReactElement => {
  if (tooltip == null) return children;

  return (
    <Tooltip.Dialog>
      {tooltip}
      {children as any}
    </Tooltip.Dialog>
  );
};
