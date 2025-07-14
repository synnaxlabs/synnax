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
  Status,
  Synnax as PSynnax,
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
  const item = List.useItem<string, ontology.Resource>(props.itemKey);
  if (item == null) return null;
  const { name, id } = item;
  const { icon, onSelect, PaletteListItem } = Ontology.useService(id.type);
  // return null if the ontology service does not have an onSelect method, that way we
  // don't show pointless items in the palette.
  return onSelect == null ? null : PaletteListItem != null ? (
    <PaletteListItem {...props} />
  ) : (
    <List.Item style={{ padding: "1.5rem" }} highlightHovered {...props}>
      <Text.WithIcon
        startIcon={isValidElement(icon) ? icon : icon(item)}
        level="p"
        weight={450}
        shade={11}
        size="medium"
      >
        {name}
      </Text.WithIcon>
    </List.Item>
  );
});

export const useResourceList = (): UseListReturn<ontology.Resource> => {
  const { data, getItem, subscribe, retrieve } = POntology.useResourceList();
  const services = Ontology.useServices();
  const client = PSynnax.use();
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
