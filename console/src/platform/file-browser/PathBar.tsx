// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.


import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";
import { segmentsOf } from "@/platform/file-browser/path";

export interface PathBarProps {
  path: string;
  onGo: (path: string) => void;
}

/** The folders on the way to the open folder. A click on one opens it. */
export const PathBar = ({ path, onGo }: PathBarProps): ReactElement => {
  const segments = segmentsOf(path);
  // A root is a place in the sidebar, so the bar leaves it out below it.
  const shown = segments.length > 1 ? segments.slice(1) : segments;
  return (
    <Flex.Box
      x
      align="center"
      empty
      bordered
      rounded
      aria-label="Path"
      className={CSS.BE("file-browser", "path")}
    >
      <Icon.Computer className={CSS.BE("file-browser", "path-icon")} />
      {shown.map((s, i) => (
        <Flex.Box x align="center" empty key={s.path} className={CSS.BE("file-browser", "segment")}>
          {i > 0 && <Icon.Caret.Right className={CSS.BE("file-browser", "separator")} />}
          <Button.Button
            variant="text"
            size="tiny"
            textColor={i === shown.length - 1 ? 11 : 9}
            className={CSS.BE("file-browser", "segment-button")}
            onClick={() => onGo(s.path)}
          >
            {s.label}
          </Button.Button>
        </Flex.Box>
      ))}
    </Flex.Box>
  );
};
