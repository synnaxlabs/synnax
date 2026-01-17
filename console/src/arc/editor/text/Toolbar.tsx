// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";
import { Breadcrumb, Flex, Icon } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { useExport } from "@/arc/export";
import { Cluster } from "@/cluster";
import { Toolbar as Base } from "@/components";
import { Export } from "@/export";
import { Layout } from "@/layout";

export interface ToolbarProps {
  layoutKey: string;
}

export const Toolbar = ({ layoutKey }: ToolbarProps): ReactElement => {
  const { name } = Layout.useSelectRequired(layoutKey);
  const handleExport = useExport();
  return (
    <Base.Header>
      <Breadcrumb.Breadcrumb level="h5">
        <Breadcrumb.Segment weight={500} color={10} level="h5">
          <Icon.Arc />
          {name}
        </Breadcrumb.Segment>
      </Breadcrumb.Breadcrumb>
      <Flex.Box x align="center" empty style={{ height: "100%", width: 66 }}>
        <Export.ToolbarButton onExport={() => void handleExport(layoutKey)} />
        <Cluster.CopyLinkToolbarButton
          name={name}
          ontologyID={arc.ontologyID(layoutKey)}
        />
      </Flex.Box>
    </Base.Header>
  );
};
