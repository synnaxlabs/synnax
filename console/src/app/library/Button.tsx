// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button as Base } from "@synnaxlabs/lyra/button";
import { Icon } from "@synnaxlabs/lyra/icon";
import type { ReactElement } from "react";

import { useOpen } from "@/app/library/tab";
import { useVisible } from "@/app/library/visible";

/** The button in the top bar that opens the Library, shown only to a user who can use
 * a section of it. */
export const Button = (): ReactElement | null => {
  const open = useOpen();
  const visible = useVisible();
  if (!visible) return null;
  return (
    <Base.Button
      size="small"
      variant="text"
      onClick={open}
      tooltip="Library"
      aria-label="Library"
    >
      <Icon.Library />
    </Base.Button>
  );
};
