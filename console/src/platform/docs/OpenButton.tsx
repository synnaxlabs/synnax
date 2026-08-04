// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Icon } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";
import { useOpenTab } from "@/platform/docs/useOpenTab";

export const OpenButton = (): ReactElement => {
  const handleOpen = useOpenTab();
  return (
    <Button.Button
      size="small"
      variant="text"
      onClick={handleOpen}
      className={CSS.BE("docs", "open-button")}
      tooltip="Open Documentation"
    >
      <Icon.QuestionMark />
    </Button.Button>
  );
};
