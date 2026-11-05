// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Panel as PPanel, Video as Base } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { useTimeRange } from "@/feature/video/useTimeRange";
import { Empty } from "@/platform/empty";
import { type Panel } from "@/platform/panel";
import { Session } from "@/session";

const Internal = (): ReactElement | null => {
  const channel = Base.useChannel();
  const range = useTimeRange();
  const visible = Session.Panel.useSelectIsTabVisible();
  const dispatch = Session.useDispatch();
  const showToolbar = () => dispatch(Session.Nav.showBottom({}));
  if (channel === 0)
    return (
      <Empty.Action
        message="No video channel"
        action="Select a channel"
        onClick={showToolbar}
      />
    );
  // A saved range shows nothing until the Core answers with its time range.
  if (range === null) return null;
  return <Base.Video visible={visible} range={range} onDoubleClick={showToolbar} />;
};

export const Video: Panel.Content = () => {
  const { key } = PPanel.useTabResource();
  return (
    <Base.Suspended videoKey={key}>
      <Internal />
    </Base.Suspended>
  );
};
