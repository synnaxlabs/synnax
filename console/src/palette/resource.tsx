// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";
import { List, Text } from "@synnaxlabs/pluto";
import { type FC, isValidElement } from "react";

import { type Ontology } from "@/ontology";

interface OntologyListItemProps extends List.ItemProps<string, ontology.Resource> {}

export const createResourceListItem = (
  ontologyServices: Ontology.Services,
): FC<OntologyListItemProps> => {
  const ResourceListItem = (props: OntologyListItemProps) => {
    const {
      entry,
      entry: { name, id },
    } = props;
    // This null check is needed because sometimes when switching to search mode from command
    // mode, the commands are passed in as resources.
    if (id == null) return null;
    const { icon, onSelect, PaletteListItem } = ontologyServices[id.type];
    // return null if the ontology service does not have an onSelect method, that way we
    // don't show pointless items in the palette.
    return onSelect == null ? null : PaletteListItem != null ? (
      <PaletteListItem {...props} />
    ) : (
      <List.ItemFrame style={{ padding: "1.5rem" }} highlightHovered {...props}>
        <Text.WithIcon
          startIcon={isValidElement(icon) ? icon : icon(entry)}
          level="p"
          weight={450}
          shade={11}
          size="medium"
        >
          {name}
        </Text.WithIcon>
      </List.ItemFrame>
    );
  };
  ResourceListItem.displayName = "ResourceListItem";
  return ResourceListItem;
};
