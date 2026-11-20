// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel, query, view } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, Panel as PPanel, Synnax, View as PView } from "@synnaxlabs/pluto";
import { strings } from "@synnaxlabs/x";
import { plural } from "pluralize";
import type { ReactElement } from "react";

import { Panel } from "@/platform/panel";

/** The explorer tab of each type of view. */
export type Explorers = Record<view.Type, Panel.Tab>;

/** The type of the tab that shows a saved view. */
export const TAB_TYPE = view.TYPE_RESOURCE_ID.type;

const useOpen = (): view.View => PView.use({ key: PPanel.useTabResource({}).key });

/**
 * Builds the tab that shows a saved view in the explorer of the view's type. The tab is
 * named "<Types> › <view>". A deleted view shows as deleted, and can be restored.
 */
export const createTab = (explorers: Explorers): Panel.Tab => {
  const Content = (): ReactElement => {
    const Explorer = explorers[useOpen().type].Content;
    return <Explorer />;
  };
  const Name: Panel.TabName = ({ allowRename = true, ...rest }) => {
    const tabKey = PPanel.useTabKey();
    const { key, type, name } = useOpen();
    const { update: rename } = PView.useRename();
    const canRename = Access.useUpdateGranted(view.resourceID(key));
    const ExplorerIcon = explorers[type].Icon;
    return (
      <>
        <ExplorerIcon />
        <Text.Text {...rest}>{`${strings.capitalize(plural(type))} ›`}</Text.Text>
        <Text.MaybeEditable
          id={Panel.tabNameID(tabKey)}
          value={name}
          overflow="fade"
          disabled={!allowRename || !canRename}
          onChange={(name) => rename({ key, name })}
          {...rest}
        />
      </>
    );
  };
  // Renders on the tab of a deleted view too, so it reads the view without suspending.
  const TabIcon: Panel.TabIcon = (props) => {
    const { data } = PView.useResult({ key: PPanel.useTabResource({}).key });
    const Glyph = data == null ? Icon.View : explorers[data.type].Icon;
    return <Glyph {...props} />;
  };
  return {
    Content,
    Name,
    Icon: TabIcon,
    Guard: Panel.createGuard({ useTombstone: PView.useTombstone }),
    restore: async ({ client, resource }) => {
      await client.views.create(
        query.requireCorpse(client.views.watch(resource.key).getCached()),
      );
    },
  };
};

export interface OpenExplorerParams {
  resourceType: view.Type;
  /** The type of the explorer tab, which shows the static view. */
  tabType: string;
}

/**
 * Returns a callback that focuses the first explorer of the type in the panel,
 * whichever view it shows. It opens one on the static view only when the panel has
 * none.
 */
export const useOpenExplorer = ({
  resourceType,
  tabType,
}: OpenExplorerParams): (() => void) => {
  const openTab = Panel.useOpenTab();
  const panelKey = Panel.useActiveKey();
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  // A tab on a saved view holds only the view's key, so the views tell which tabs are
  // explorers of the type.
  const retrieveExplorers = async (): Promise<Set<view.Key>> => {
    const cached =
      panelKey == null ? undefined : client?.panels.watch(panelKey).getCached();
    if (client == null || !query.isLive(cached)) return new Set();
    const keys = panel
      .tabs(cached.root)
      .flatMap((tab) =>
        tab.variant === "resource" && tab.resource.type === TAB_TYPE
          ? [tab.resource.key]
          : [],
      );
    if (keys.length === 0) return new Set();
    // A query on keys alone answers from the cache and fetches only the views it lacks.
    const views = await client.views.retrieve({ where: { key: { in: keys } } });
    return new Set(
      views.filter(({ type }) => type === resourceType).map(({ key }) => key),
    );
  };
  return () =>
    handleError(async () => {
      const explorers = await retrieveExplorers();
      openTab(
        { variant: "view", type: tabType },
        {
          focus: (tab) =>
            tab.variant === "view"
              ? tab.type === tabType
              : tab.resource.type === TAB_TYPE && explorers.has(tab.resource.key),
        },
      );
    }, `Failed to open ${resourceType} explorer`);
};
