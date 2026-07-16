// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Base } from "@/arc/graph/node/Base";
import { type Config } from "@/arc/graph/node/constant/config";
import { type types } from "@/arc/graph/node/types";
import { Icon } from "@/icon";
import { Text } from "@/text";

export type SymbolProps = types.SymbolProps<Config>;

export const Constant = ({ config: { value }, scale }: SymbolProps) => (
  <Base
    type="Constant"
    Icon={<Icon.Constant />}
    color="var(--pluto-success-z-20)"
    textColor="var(--pluto-success-z)"
    sources={[{ key: "output", Icon: Icon.Value }]}
    scale={scale}
  >
    <Text.Text level="h4" weight={500} variant="code">
      {value.toString()}
    </Text.Text>
  </Base>
);
