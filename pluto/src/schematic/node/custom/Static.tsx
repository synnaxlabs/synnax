// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Note } from "@synnaxlabs/lyra/note";
import { Text } from "@synnaxlabs/lyra/text";
import { type location } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Custom } from "@/schematic/node/common/custom";
import { Primitive } from "@/schematic/node/common/primitive";
import { Symbol } from "@/schematic/symbol";

export interface StaticProps {
  specKey: string;
  orientation?: location.Outer;
  scale?: number;
  className?: string;
  stateOverrides?: schematic.symbol.StateOverride[];
}

export const Static = ({
  specKey,
  orientation = "left",
  scale = 1,
  className,
  stateOverrides,
}: StaticProps): ReactElement => {
  const { symbol, missing } = Symbol.useResolved(specKey);
  const spec = symbol?.data;
  const paints = Custom.usePaints(spec, 0, stateOverrides);
  const invalid = spec != null && !Custom.isValidSVG(spec.drawing.markup);
  if (missing || invalid)
    return (
      <Note.Note variant="warning" className={className}>
        <Text.Text level="p" status="warning">
          <Icon.Warning />
          {missing ? "Missing custom symbol" : "Invalid custom symbol"}
        </Text.Text>
      </Note.Note>
    );
  return (
    <Primitive.Div
      orientation={orientation}
      className={CSS.cls(CSS.BM("symbol", "custom"), CSS.B("custom-static"), className)}
    >
      {spec != null && (
        <Custom.Drawing
          spec={spec}
          orientation={orientation}
          scale={scale}
          paints={paints}
        />
      )}
    </Primitive.Div>
  );
};
