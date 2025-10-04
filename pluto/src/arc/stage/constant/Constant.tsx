// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import z from "zod/v4";

import { Base } from "@/arc/stage/Base";
import { type types } from "@/arc/stage/types";
import { Icon } from "@/icon";
import { Text } from "@/text";

const constant = z.object({
  value: z.number(),
});

export const configZ = constant;

export type Config = z.infer<typeof configZ>;

export type SymbolProps = types.SymbolProps<Config>;

export const Constant = ({ value, scale }: SymbolProps) => (
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
