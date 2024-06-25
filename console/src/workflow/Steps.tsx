// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { FC, ReactElement, useState } from "react";

export interface StepProps {
  next: () => void;
}

export type Step = FC<StepProps>;

export type Steps = Record<string, Step>;

export interface WorkflowProps {
  start: string;
  steps: Steps;
}

export const Workflow = ({ steps, start }: WorkflowProps): ReactElement => {
  const [step, setStep] = useState<string>(start);
  const Step = steps[step];
  return <Step key={step} next={setStep} />;
};
