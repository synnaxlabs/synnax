import { ReactElement } from "react";

import { OntologyID } from "@synnaxlabs/client";
import type { OntologyResourceType } from "@synnaxlabs/client";
import { AiFillDatabase } from "react-icons/ai";
import { MdOutlineDeviceHub, MdSensors } from "react-icons/md";

import { LinePlotVisualization } from "../visualization/types";

import { LayoutPlacer } from "@/features/layout";
import { createVisualization } from "@/features/visualization";

export interface ResourceType {
  type: OntologyResourceType;
  icon: ReactElement;
  onSelect?: (id: OntologyID) => void;
  hasChildren: boolean;
}

export const resourceTypes = (
  placer: LayoutPlacer
): Record<OntologyResourceType, ResourceType> => ({
  builtin: {
    type: "builtin",
    icon: <AiFillDatabase />,
    hasChildren: true,
  },
  cluster: {
    type: "cluster",
    icon: <AiFillDatabase />,
    hasChildren: true,
  },
  node: {
    type: "node",
    icon: <MdOutlineDeviceHub />,
    hasChildren: true,
  },
  channel: {
    type: "channel",
    icon: <MdSensors />,
    hasChildren: false,
    onSelect: (id) => {
      placer(
        createVisualization<LinePlotVisualization>({ channels: [id.key], ranges: [] })
      );
    },
  },
});
