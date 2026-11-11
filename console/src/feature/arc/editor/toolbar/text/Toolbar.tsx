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
import { Breadcrumb } from "@synnaxlabs/lyra/breadcrumb";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Text } from "@synnaxlabs/lyra/text";
import { Arc } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Arc as PlatformArc } from "@/platform/arc";
import { Core } from "@/platform/core";
import { CSS } from "@/platform/css";
import { Export } from "@/platform/export";
import { Toolbar as Base } from "@/platform/toolbar";

export const Toolbar = (): ReactElement => {
  const key = Arc.useKey();
  const name = Arc.useName({ key });
  const TypeIcon = PlatformArc.TYPE_ICONS[Arc.useType()];
  return (
    <Base.Content>
      <Base.Header>
        <Breadcrumb.Breadcrumb level="h5">
          <Base.TitleSegment>
            <TypeIcon />
            {name}
          </Base.TitleSegment>
        </Breadcrumb.Breadcrumb>
        <Flex.Box
          x
          align="center"
          gap="small"
          className={CSS.BE("arc-toolbar", "actions")}
        >
          <Export.ToolbarButton id={arc.resourceID(key)} />
          <Core.CopyLinkToolbarButton name={name} resourceID={arc.resourceID(key)} />
        </Flex.Box>
      </Base.Header>
      <Flex.Box center>
        <Text.Text status="disabled">No content</Text.Text>
      </Flex.Box>
    </Base.Content>
  );
};
