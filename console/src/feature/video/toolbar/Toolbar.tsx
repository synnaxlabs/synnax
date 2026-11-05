// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { video } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Panel as PPanel, Video } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Download } from "@/feature/video/toolbar/Download";
import { Properties } from "@/feature/video/toolbar/Properties";
import { Core } from "@/platform/core";
import { type Panel } from "@/platform/panel";
import { Toolbar as Base } from "@/platform/toolbar";

const Internal = (): ReactElement => {
  const name = Video.useName();
  const key = Video.useKey();
  return (
    <Base.Content>
      <Base.Header>
        <Base.Title>
          <Icon.Video />
          {name}
        </Base.Title>
        <Base.Actions>
          <Download />
          <Core.CopyLinkToolbarButton name={name} ontologyID={video.ontologyID(key)} />
        </Base.Actions>
      </Base.Header>
      <Base.Body>
        <Properties />
      </Base.Body>
    </Base.Content>
  );
};

export const Toolbar: Panel.Toolbar = () => {
  const { key } = PPanel.useTabResource();
  return (
    <Video.Suspended videoKey={key}>
      <Internal />
    </Video.Suspended>
  );
};
