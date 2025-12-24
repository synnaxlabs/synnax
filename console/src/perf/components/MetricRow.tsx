// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/perf/components/MetricRow.css";

import { Flex, Text } from "@synnaxlabs/pluto";
import { memo, type ReactElement, type ReactNode } from "react";

import { WithTooltip } from "@/perf/components/WithTooltip";
import { type DisplayStatus } from "@/perf/ui-types";

interface MetricRowProps {
  label: string;
  value: string | ReactNode;
  status?: DisplayStatus;
  tooltip?: string;
}

const MetricRowImpl = ({
  label,
  value,
  status,
  tooltip,
}: MetricRowProps): ReactElement => (
  <WithTooltip tooltip={tooltip}>
    <Flex.Box
      x
      justify="between"
      align="center"
      className="console-perf-row"
      tabIndex={0}
    >
      <Text.Text level="small" className="console-perf-row-label">
        {label}
      </Text.Text>
      <Text.Text level="small" className="console-perf-row-value" data-status={status}>
        {value}
      </Text.Text>
    </Flex.Box>
  </WithTooltip>
);

export const MetricRow = memo(MetricRowImpl);
