import { type ReactElement } from "react";

import { type Input, Steps as PSteps } from "@synnaxlabs/pluto";

export interface StepsProps extends Input.Control<string> {}

const STEPS: PSteps.Step[] = [
  {
    key: "properties",
    title: "Define Properties",
  },
  {
    key: "physicalPlan",
    title: "Create Channels",
  },
  {
    key: "confirm",
    title: "Confirm",
  },
  {
    key: "nextSteps",
    title: "Next Steps",
  },
];

export const Steps = ({ value, onChange }: StepsProps): ReactElement => (
  <PSteps.Steps steps={STEPS} value={value} onChange={onChange} />
);
