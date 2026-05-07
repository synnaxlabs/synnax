// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Aether } from "@synnaxlabs/charon/aether";
import { Key } from "@synnaxlabs/charon/key";
import { type ReactElement, useEffect } from "react";

import { useContext } from "@/lineplot/LinePlot";
import { Line as Base } from "@/vis/line";

export interface LineProps extends Base.LineProps, Aether.ComponentProps {
  legendGroup: string;
}

export const Line = ({
  aetherKey,
  color,
  label = "",
  legendGroup,
  visible = true,
  ...rest
}: LineProps): ReactElement => {
  const cKey = Key.useUnique(aetherKey);
  const { setLine, removeLine } = useContext("Line");
  useEffect(() => {
    setLine({ key: cKey, color, label, visible, legendGroup });
    return () => removeLine(cKey);
  }, [label, color, visible, legendGroup]);
  return (
    <Base.Line
      aetherKey={cKey}
      color={color}
      label={label}
      visible={visible}
      {...rest}
    />
  );
};
