import { effect } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/pluto";

import { Effect } from "@/effect";
import { Ontology } from "@/ontology";

const handleSelect: Ontology.HandleSelect = ({ selection, placeLayout }) => {
  selection.forEach((s) => {
    placeLayout(
      Effect.createEditLayout({
        key: s.id.key,
      }),
    );
  });
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  ...Ontology.NOOP_SERVICE,
  type: "effect",
  icon: <Icon.Effect />,
  canDrop: () => true,
  onSelect: handleSelect,
};
