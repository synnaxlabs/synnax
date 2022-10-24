import { OntologyID, OntologyResourceType } from "@synnaxlabs/client";
import { ReactElement } from "react";
import { AiFillDatabase } from "react-icons/ai";
import { MdOutlineDeviceHub, MdSensors } from "react-icons/md";

export interface ResourceType {
  type: OntologyResourceType;
  icon: ReactElement;
  onSelect?: (id: OntologyID) => void;
  hasChildren: boolean;
}

export const resourceTypes: Record<OntologyResourceType, ResourceType> = {
  [OntologyResourceType.Cluster]: {
    type: OntologyResourceType.Cluster,
    icon: <AiFillDatabase />,
    hasChildren: true,
  },
  [OntologyResourceType.Node]: {
    type: OntologyResourceType.Node,
    icon: <MdOutlineDeviceHub />,
    hasChildren: true,
  },
  [OntologyResourceType.Channel]: {
    type: OntologyResourceType.Channel,
    icon: <MdSensors />,
    hasChildren: false,
  },
};
