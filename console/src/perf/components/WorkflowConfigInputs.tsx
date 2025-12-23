// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex, Input } from "@synnaxlabs/pluto";
import { type bounds } from "@synnaxlabs/x";
import { memo, type ReactElement } from "react";

import { type WorkflowConfig } from "@/perf/workflows/types";

const ITERATIONS_BOUNDS: bounds.Bounds = { lower: 1, upper: Infinity };
const DELAY_BOUNDS: bounds.Bounds = { lower: 0, upper: 10000 };

export interface WorkflowConfigInputsProps {
  config: WorkflowConfig;
  onChange: (config: WorkflowConfig) => void;
  disabled?: boolean;
}

const WorkflowConfigInputsImpl = ({
  config,
  onChange,
  disabled,
}: WorkflowConfigInputsProps): ReactElement => {
  const handleIterationsChange = (iterations: number) => {
    onChange({ ...config, iterations });
  };

  const handleDelayBetweenIterationsChange = (delayBetweenIterationsMs: number) => {
    onChange({ ...config, delayBetweenIterationsMs });
  };

  const handleDelayBetweenWorkflowsChange = (delayBetweenWorkflowsMs: number) => {
    onChange({ ...config, delayBetweenWorkflowsMs });
  };

  return (
    <Flex.Box y gap="small">
      <Input.Item label="Iterations">
        <Input.Numeric
          value={config.iterations}
          onChange={handleIterationsChange}
          bounds={ITERATIONS_BOUNDS}
          disabled={disabled}
        />
      </Input.Item>
      <Input.Item label="Delay Between Iterations (ms)">
        <Input.Numeric
          value={config.delayBetweenIterationsMs}
          onChange={handleDelayBetweenIterationsChange}
          bounds={DELAY_BOUNDS}
          disabled={disabled}
        />
      </Input.Item>
      <Input.Item label="Delay Between Macros (ms)">
        <Input.Numeric
          value={config.delayBetweenWorkflowsMs}
          onChange={handleDelayBetweenWorkflowsChange}
          bounds={DELAY_BOUNDS}
          disabled={disabled}
        />
      </Input.Item>
    </Flex.Box>
  );
};

export const WorkflowConfigInputs = memo(WorkflowConfigInputsImpl);
