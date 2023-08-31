// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import { Layout } from "@/layout";
import { NavControls as LineNavControls } from "@/line/NavControls";
import { NavControls as PidNavControls } from "@/pid/NavControls";

export const NavControls = (): ReactElement => {
  const layout = Layout.useSelectActiveMosaicLayout();

  switch (layout?.type) {
    case "line":
      return <LineNavControls />;
    case "pid":
      return <PidNavControls />;
    default:
      return <></>;
  }
};
