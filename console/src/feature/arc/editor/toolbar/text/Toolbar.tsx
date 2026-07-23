// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/arc/editor/toolbar/text/Toolbar.css";

import { arc } from "@synnaxlabs/client";
import { Arc, Breadcrumb, Flex, Icon, Text } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { useExport } from "@/feature/arc/export";
import { Cluster } from "@/platform/cluster";
import { CSS } from "@/platform/css";
import { Export } from "@/platform/export";
import { Toolbar as Base } from "@/platform/toolbar";

export const Toolbar = (): ReactElement => {
  const key = Arc.useKey();
  const name = Arc.useSelectName(key);
  const handleExport = useExport();
  return (
    <>
      <Base.Header>
        <Breadcrumb.Breadcrumb level="h5">
          <Breadcrumb.Segment weight={500} color={10} level="h5">
            <Icon.Arc />
            {name}
          </Breadcrumb.Segment>
        </Breadcrumb.Breadcrumb>
        <Flex.Box x align="center" empty className={CSS.BE("arc-toolbar", "actions")}>
          <Export.ToolbarButton onExport={() => handleExport(key)} />
          <Cluster.CopyLinkToolbarButton name={name} ontologyID={arc.ontologyID(key)} />
        </Flex.Box>
      </Base.Header>
      <Flex.Box center>
        <Text.Text status="disabled">No Content</Text.Text>
      </Flex.Box>
    </>
  );
};
