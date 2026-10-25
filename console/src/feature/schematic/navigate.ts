// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { query, type schematic, type Synnax as Client } from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/lyra/status";
import { Synnax } from "@synnaxlabs/pluto";

import { Panel } from "@/platform/panel";
import { Session } from "@/session";

interface PageTarget {
  noun: string;
  retrieve: (client: Client, key: string) => Promise<unknown>;
}

const PAGE_TARGETS: Record<schematic.PageType, PageTarget> = {
  schematic: {
    noun: "schematic",
    retrieve: (client, key) => client.schematics.retrieve(key),
  },
  lineplot: {
    noun: "line plot",
    retrieve: (client, key) => client.lineplots.retrieve(key),
  },
  log: {
    noun: "log",
    retrieve: (client, key) => client.logs.retrieve(key),
  },
  table: {
    noun: "table",
    retrieve: (client, key) => client.tables.retrieve(key),
  },
};

type NodeClickHandler = (nodeId: string, dblClick: boolean) => void;

export const useHandleNodeClickAction = (schematicKey: string): NodeClickHandler => {
  const client = Synnax.use();
  const getSchematic = Session.Schematic.useGet();
  const handleError = Status.useErrorHandler();
  const openTab = Panel.useOpenTab();

  return (nodeId: string, dblClick: boolean) => {
    const ui = getSchematic({ key: schematicKey });
    if (ui == null || ui.editable || client == null) return;
    const cached = client.schematics.getCached(schematicKey);
    const config = query.isLive(cached) ? cached.configs?.[nodeId] : undefined;
    if (config?.variant !== "off_page_reference") return;
    const { page } = config;
    if (page == null || page.key.length === 0) return;
    const navigatesOnDblClick = !config.dblClickNavDisabled;
    if (dblClick !== navigatesOnDblClick) return;
    const target = PAGE_TARGETS[page.type];
    const { label } = config.label;
    handleError(
      async () => {
        await target.retrieve(client, page.key);
        openTab({ variant: "resource", resource: page });
      },
      { message: `Failed to open ${target.noun}`, name: label },
    );
  };
};
