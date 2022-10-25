import { OntologyID, OntologyResourceType } from "@synnaxlabs/client";
import { Dispatch, ReactElement } from "react";
import { AiFillDatabase } from "react-icons/ai";
import { MdOutlineDeviceHub, MdSensors } from "react-icons/md";
import { insertTab } from "../../mosaic/slice";

export interface ResourceType {
  type: OntologyResourceType;
  icon: ReactElement;
  onSelect?: (id: OntologyID) => void;
  hasChildren: boolean;
}

export const resourceTypes = (
  dispatch: Dispatch<any>
): Record<OntologyResourceType, ResourceType> => ({
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
    onSelect: (id) =>
      dispatch(
        insertTab({
          tab: {
            tabKey: `channel ${id.toString()}`,
            title: `CdA_LOX_inj`,
          },
        })
      ),
  },
});
