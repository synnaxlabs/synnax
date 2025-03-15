// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Fragment, type ReactElement } from "react";

import { Align } from "@/align";
import { Button } from "@/button";
import { CSS } from "@/css";
import { type Input } from "@/input";

export interface Step {
  key: string;
  title: string;
}

export interface StepsProps
  extends Omit<Align.SpaceProps, "children" | "onChange">,
    Input.Control<string> {
  steps: Step[];
}

export const Steps = ({
  steps,
  value,
  onChange,
  ...rest
}: StepsProps): ReactElement => {
  const selectedIdx = steps.findIndex((step) => step.key === value);
  return (
    <Align.Space direction="x" align="center" className={CSS.B("steps")} {...rest}>
      {steps.map((step, i) => (
        <Fragment key={step.key}>
          <Button.Button
            onClick={() => onChange(step.key)}
            key={step.key}
            variant="outlined"
            disabled={i > selectedIdx}
          >
            {i + 1}. {step.title}
          </Button.Button>
          {i !== steps.length - 1 && (
            <Icon.Arrow.Right
              style={{ fontSize: "15px", color: "var(--pluto-gray-l8)" }}
            />
          )}
        </Fragment>
      ))}
    </Align.Space>
  );
};
