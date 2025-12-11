import { type ReactElement } from "react";

import { useContext } from "@/timeline/context";

export const Axis = (): ReactElement => {
  const { bounds, viewport } = useContext("Timeline.Axis");
  return <div>Axis</div>;
};
