// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology } from "@synnaxlabs/client";
import {
  Component,
  List,
  Ontology as POntology,
  Select,
  Status,
  Synnax,
  Text,
} from "@synnaxlabs/pluto";
import { isValidElement, useCallback } from "react";
import { useStore } from "react-redux";

import { Layout } from "@/layout";
import { Ontology } from "@/ontology";
import { type UseListReturn } from "@/palette/list";
import { type RootAction, type RootState } from "@/store";

interface OntologyListItemProps extends List.ItemProps<string> {}

const listItem = Component.renderProp((props: OntologyListItemProps) => {
  const { itemKey } = props;
  const id = ontology.idZ.parse(itemKey);
  const item = List.useItem<string, ontology.Resource>(itemKey);
  const { icon, onSelect, PaletteListItem } = Ontology.useService(id.type);
  if (item == null) return null;
  const { name } = item;
  // return null if the ontology service does not have an onSelect method, that way we
  // don't show pointless items in the palette.
  return onSelect == null ? null : PaletteListItem != null ? (
    <PaletteListItem {...props} />
  ) : (
    <Select.ListItem highlightHovered {...props}>
      <Text.Text weight={450} gap="medium">
        {icon != null && (isValidElement(icon) ? icon : icon(item))}
        {name}
      </Text.Text>
    </Select.ListItem>
  );
});

export const useResourceList = (): UseListReturn<ontology.Resource> => {
  const { data, getItem, subscribe, retrieve } = POntology.useResourceList();
  const services = Ontology.useServices();
  const client = Synnax.use();
  const store = useStore<RootState, RootAction>();
  const addStatus = Status.useAdder();
  const placeLayout = Layout.usePlacer();
  const removeLayout = Layout.useRemover();
  const handleError = Status.useErrorHandler();

  const handleSelect = useCallback(
    (key: string) => {
      if (client == null) return;
      const { type } = ontology.idZ.parse(key);
      services[type].onSelect?.({
        services,
        store,
        addStatus,
        placeLayout,
        removeLayout,
        handleError,
        client,
        selection: getItem([key]),
      });
    },
    [client, services, store, addStatus, placeLayout, removeLayout, handleError],
  );

  return { data, getItem, subscribe, handleSelect, listItem, retrieve };
};
