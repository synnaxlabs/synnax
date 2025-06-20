// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Icon } from "@synnaxlabs/pluto";

export interface TareButtonProps {
  disabled?: boolean;
  onTare: () => void;
}

export const TareButton = ({ onTare, disabled }: TareButtonProps) => (
  <Button.Icon
    variant="outlined"
    disabled={disabled}
    onClick={(e) => {
      e.stopPropagation();
      onTare();
    }}
    size="small"
    tooltip="Tare"
  >
    <Icon.Tare />
  </Button.Icon>
);
