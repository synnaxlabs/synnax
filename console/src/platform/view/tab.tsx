// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type panel, query, view } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, Panel as PPanel, Synnax, View as PView } from "@synnaxlabs/pluto";
import { caseconv } from "@synnaxlabs/x";
import { plural } from "pluralize";
import { type ReactElement } from "react";

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
        <Text.Text {...rest}>{`${caseconv.capitalize(plural(type))} ›`}</Text.Text>
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
 * Returns a callback that focuses the panel's explorer of the type, whichever view it
 * shows. It opens one on the static view only when the panel has none.
 */
export const useOpenExplorer = ({
  resourceType,
  tabType,
}: OpenExplorerParams): (() => void) => {
  const openTab = Panel.useOpenTab();
  const client = Synnax.use();
  // The tab name of a saved view loads it, so its type is in the cache.
  const isExplorer = (tab: panel.Tab): boolean => {
    if (tab.variant === "view") return tab.type === tabType;
    if (tab.resource.type !== TAB_TYPE) return false;
    const cached = client?.views.watch(tab.resource.key).getCached();
    return query.isLive(cached) && cached.type === resourceType;
  };
  return () => openTab({ variant: "view", type: tabType }, { focus: isExplorer });
};
