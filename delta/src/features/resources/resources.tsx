import { OntologyID, OntologyResourceType } from "@synnaxlabs/client";
import { Dispatch, ReactElement } from "react";
import { AiFillDatabase } from "react-icons/ai";
import { MdOutlineDeviceHub, MdSensors } from "react-icons/md";
import { LayoutPlacer } from "@/features/layout";
import { createVisualization } from "@/features/visualization";
import { LinePlotVisualization } from "../visualization/types";

export interface ResourceType {
	type: OntologyResourceType;
	icon: ReactElement;
	onSelect?: (id: OntologyID) => void;
	hasChildren: boolean;
}

export const resourceTypes = (
	placer: LayoutPlacer
): Record<OntologyResourceType, ResourceType> => ({
	[OntologyResourceType.Builtin]: {
		type: OntologyResourceType.Builtin,
		icon: <AiFillDatabase />,
		hasChildren: true,
	},
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
		onSelect: (id) => {
			placer(
				createVisualization<LinePlotVisualization>({ channels: [id.key], ranges: [] })
			);
		},
	},
});
