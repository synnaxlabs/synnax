// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/fs/InputPath.css";

import { Button, Flex, Icon, type Input, Status } from "@synnaxlabs/pluto";
import { primitive } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";
import { Runtime } from "@/platform/runtime";

export interface InputPathProps
  extends Input.Control<string>, Omit<Flex.BoxProps, "value" | "onChange"> {
  filters?: Runtime.FileFilter[];
}

/**
 * An input holding an absolute file path chosen through the native picker. Only the
 * desktop app can produce a path, so the picker rejects in the browser.
 */
export const InputPath = ({
  value,
  onChange,
  filters,
  ...rest
}: InputPathProps): ReactElement => {
  const path = value;
  const handleError = Status.useErrorHandler();
  const handleClick = () =>
    handleError(async () => {
      const path = await Runtime.pickPath({ filters });
      if (path == null) return;
      onChange(path);
    }, "Failed to open file");
  return (
    <Flex.Box pack className={CSS.B("input-path")} borderColor={6} {...rest}>
      <Button.Button
        level="small"
        className={CSS.B("path")}
        variant="outlined"
        grow
        onClick={handleClick}
        size="medium"
        textColor={9}
        weight={450}
      >
        {primitive.isNonZero(path) ? (
          <>
            <Icon.Attachment className={CSS.BE("input-path", "icon")} />
            {path}
          </>
        ) : (
          "No file selected"
        )}
      </Button.Button>
      <Button.Button
        variant="outlined"
        className={CSS.B("select")}
        onClick={handleClick}
      >
        Select file
      </Button.Button>
    </Flex.Box>
  );
};
