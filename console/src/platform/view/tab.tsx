// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { query, view } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Text } from "@synnaxlabs/lyra/text";
import { Panel as PPanel, View as PView } from "@synnaxlabs/pluto";
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
  const Name: Panel.TabName = ({ level, weight, color }) => {
    const { type, name } = useOpen();
    const ExplorerIcon = explorers[type].Icon;
    return (
      <>
        <ExplorerIcon />
        <Text.Text overflow="fade" level={level} weight={weight} color={color}>
          {`${caseconv.capitalize(plural(type))} › ${name}`}
        </Text.Text>
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
