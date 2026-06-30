// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Form, Header, Icon } from "@synnaxlabs/pluto";
import { binary } from "@synnaxlabs/x";
import { useCallback } from "react";

export interface DetailsHeaderProps {
  path: string;
  disabled?: boolean;
}

export const DetailsHeader = ({ path, disabled = false }: DetailsHeaderProps) => {
  const { get } = Form.useContext();
  const getText = useCallback(
    () => binary.JSON_CODEC.encodeString(get(path).value),
    [get, path],
  );
  return (
    <Header.Header>
      <Header.Title weight={500} wrap={false} color={10}>
        Details
      </Header.Title>
      <Header.Actions>
        <Button.Copy
          disabled={disabled}
          tooltip="Copy details as JSON"
          tooltipLocation="left"
          variant="text"
          text={getText}
          successMessage="Copied details to clipboard"
          contrast={2}
          textColor={9}
        >
          <Icon.JSON />
        </Button.Copy>
      </Header.Actions>
    </Header.Header>
  );
};
