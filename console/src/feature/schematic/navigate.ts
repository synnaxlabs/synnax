// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  lineplot,
  log,
  type ontology,
  query,
  schematic,
  type Synnax as Client,
  table,
} from "@synnaxlabs/client";
import { Schematic, Status, Synnax } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Panel } from "@/platform/panel";
import { Session } from "@/session";

interface PageTarget {
  noun: string;
  retrieve: (client: Client, key: string) => Promise<unknown>;
  ontologyID: (key: string) => ontology.ID;
}

const PAGE_TARGETS: Record<Schematic.Node.OffPageReference.PageType, PageTarget> = {
  schematic: {
    noun: "Schematic",
    retrieve: (client, key) => client.schematics.retrieve(key),
    ontologyID: schematic.ontologyID,
  },
  lineplot: {
    noun: "Line plot",
    retrieve: (client, key) => client.lineplots.retrieve(key),
    ontologyID: lineplot.ontologyID,
  },
  log: {
    noun: "Log",
    retrieve: (client, key) => client.logs.retrieve(key),
    ontologyID: log.ontologyID,
  },
  table: {
    noun: "Table",
    retrieve: (client, key) => client.tables.retrieve(key),
    ontologyID: table.ontologyID,
  },
};

type NodeClickHandler = (nodeId: string, dblClick: boolean) => void;

export const useHandleNodeClickAction = (schematicKey: string): NodeClickHandler => {
  const client = Synnax.use();
  const getSchematic = Session.Schematic.useGet();
  const handleError = Status.useErrorHandler();
  const openTab = Panel.useOpenTab();

  return useCallback(
    (nodeId: string, dblClick: boolean) => {
      const ui = getSchematic({ key: schematicKey });
      if (ui == null || ui.editable || client == null) return;
      const cached = client.schematics.getCached(schematicKey);
      const config = query.isLive(cached) ? cached.configs?.[nodeId] : undefined;
      if (
        config?.variant !== "offPageReference" ||
        typeof config.page !== "string" ||
        config.page.length === 0
      )
        return;
      const dblClickNav = config.dblClickNav !== false;
      if (dblClick !== dblClickNav) return;
      const { type, key } = Schematic.Node.OffPageReference.parsePage(config.page);
      const target = PAGE_TARGETS[type];
      const labelObj = config.label as { label?: string } | undefined;
      const label = labelObj?.label;
      const name =
        label != null && label.length > 0
          ? label
          : `Referenced ${target.noun.toLowerCase()}`;
      handleError(async () => {
        await target.retrieve(client, key);
        openTab({ variant: "resource", resource: target.ontologyID(key) });
      }, `${target.noun} "${name}" not found`);
    },
    [getSchematic, schematicKey, openTab, handleError, client],
  );
};
