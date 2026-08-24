// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Flex, Form, Header, Icon } from "@synnaxlabs/pluto";
import { binary } from "@synnaxlabs/x";
import { type ReactNode, useCallback } from "react";

import { CSS } from "@/platform/css";

export interface DetailsHeaderProps {
  path: string;
  disabled?: boolean;
  /** Names the selected item; shown in place of "Details" while one is selected. */
  children?: ReactNode;
  /** Rendered before the title. */
  start?: ReactNode;
}

export const DetailsHeader = ({
  path,
  disabled = false,
  children,
  start,
}: DetailsHeaderProps) => {
  const { get } = Form.useContext();
  const getText = useCallback(
    () => binary.JSON_CODEC.encodeString(get(path).value),
    [get, path],
  );
  return (
    <Header.Header>
      <Flex.Box x align="center" gap="small" className={CSS.BE("panes", "title")}>
        {start}
        <Header.Title weight={500} wrap={false} color={10}>
          {disabled || children == null ? "Details" : children}
        </Header.Title>
      </Flex.Box>
      <Header.Actions>
        <Button.Copy
          disabled={disabled}
          tooltip="Copy details as JSON"
          tooltipLocation="left"
          variant="text"
          size="small"
          text={getText}
          successMessage="Copied details to clipboard"
          textColor={9}
        >
          <Icon.JSON />
        </Button.Copy>
      </Header.Actions>
    </Header.Header>
  );
};
