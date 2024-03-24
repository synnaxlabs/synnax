import { type ReactElement } from "react";

import { type Input, Steps as PSteps } from "@synnaxlabs/pluto";

export interface StepsProps extends Input.Control<string> {}

const TABS: PSteps.Step[] = [
  {
    key: "properties",
    title: "Properties",
  },
  {
    key: "physicalPlan",
    title: "Channel Creation",
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
  <PSteps.Steps steps={TABS} value={value} onChange={onChange} />
);
