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
    key: "softwareTasks",
    title: "Software Tasks",
  },
];

export const Steps = ({ value, onChange }: StepsProps): ReactElement => (
  <PSteps.Steps steps={TABS} value={value} onChange={onChange} />
);
