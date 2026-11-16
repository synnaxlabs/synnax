// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type view } from "@synnaxlabs/client";
import { Text } from "@synnaxlabs/lyra/text";
import { Panel as PPanel, View as PView } from "@synnaxlabs/pluto";
import { caseconv } from "@synnaxlabs/x";
import { plural } from "pluralize";

import { type Panel } from "@/platform/panel";
import { Session } from "@/session";

export interface CreateTabNameParams extends Panel.CreateStaticTabNameParams {
  resourceType: view.Type;
}

/**
 * Builds the name of an explorer tab. The tab takes the name given while it shows its
 * static view, and "<Types> › <view>" while it shows a saved one.
 */
export const createTabName = ({
  resourceType,
  name,
  icon,
}: CreateTabNameParams): Panel.TabName => {
  const prefix = caseconv.capitalize(plural(resourceType));
  const Name: Panel.TabName = ({ level, weight, color }) => {
    const open = Session.View.useSelectOpen(PPanel.useTabKey());
    const saved = PView.useResult(open == null ? null : { key: open });
    const viewName = saved.variant === "error" ? undefined : saved.data?.name;
    return (
      <>
        {icon}
        <Text.Text overflow="fade" level={level} weight={weight} color={color}>
          {viewName == null ? name : `${prefix} › ${viewName}`}
        </Text.Text>
      </>
    );
  };
  return Name;
};
