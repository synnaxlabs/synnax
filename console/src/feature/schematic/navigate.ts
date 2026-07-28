// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { query, schematic } from "@synnaxlabs/client";
import { Status, Synnax } from "@synnaxlabs/pluto";
import { useCallback, useMemo } from "react";

import { Panel } from "@/platform/panel";
import { Session } from "@/session";

type SchematicRetriever = (key: string) => Promise<schematic.Schematic>;

const navigateToLinkedSchematic = async (
  retrieve: SchematicRetriever,
  page: string,
  openTab: Panel.OpenTab,
): Promise<void> => {
  const s = await retrieve(page);
  openTab({ variant: "resource", resource: schematic.ontologyID(s.key) });
};

type NodeClickHandler = (nodeId: string, dblClick: boolean) => void;

export const useHandleNodeClickAction = (schematicKey: string): NodeClickHandler => {
  const client = Synnax.use();
  const getSchematic = Session.Schematic.useGet();
  const retrieve: SchematicRetriever | null = useMemo(
    () =>
      client != null ? (key: string) => client.schematics.retrieve({ key }) : null,
    [client],
  );
  const handleError = Status.useErrorHandler();
  const openTab = Panel.useOpenTab();

  return useCallback(
    (nodeId: string, dblClick: boolean) => {
      const ui = getSchematic({ key: schematicKey });
      if (ui == null || ui.editable || retrieve == null) return;
      const cached = client?.schematics.getCached({ key: schematicKey });
      const config =
        cached != null && !query.Deleted.matches(cached)
          ? cached.configs?.[nodeId]
          : undefined;
      if (
        config?.variant !== "offPageReference" ||
        typeof config.page !== "string" ||
        config.page.length === 0
      )
        return;
      const dblClickNav = config.dblClickNav !== false;
      if (dblClick !== dblClickNav) return;
      const { page } = config;
      const labelObj = config.label as { label?: string } | undefined;
      const label = labelObj?.label;
      const name = label != null && label.length > 0 ? label : "Referenced schematic";
      handleError(
        () => navigateToLinkedSchematic(retrieve, page, openTab),
        `Schematic "${name}" not found`,
      );
    },
    [getSchematic, schematicKey, retrieve, openTab, handleError, client],
  );
};
