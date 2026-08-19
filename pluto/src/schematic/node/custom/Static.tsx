// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { type location } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Icon } from "@/icon";
import { Note } from "@/note";
import { Custom } from "@/schematic/node/common/custom";
import { Handle } from "@/schematic/node/common/handle";
import { Primitive } from "@/schematic/node/common/primitive";
import { Symbol } from "@/schematic/symbol";
import { Text } from "@/text";

export interface StaticProps {
  specKey: string;
  orientation?: location.Outer;
  scale?: number;
  className?: string;
  stateOverrides?: schematic.symbol.State[];
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
  const setContainer = Custom.useRender({
    orientation,
    activeState: "base",
    externalScale: scale,
    spec,
    stateOverrides,
  });
  if (missing)
    return (
      <Note.Note variant="warning" className={className}>
        <Text.Text level="p" status="warning">
          <Icon.Warning />
          Missing custom symbol
        </Text.Text>
      </Note.Note>
    );
  const handles = spec?.handles ?? [];
  return (
    <Primitive.Div
      ref={setContainer}
      orientation={orientation}
      className={CSS.cls(CSS.BM("symbol", "custom"), CSS.B("custom-static"), className)}
    >
      <Handle.Boundary orientation={orientation}>
        {handles.map((handle) => (
          <Handle.Handle
            key={handle.key}
            id={handle.key}
            location={handle.orientation}
            orientation={orientation}
            left={handle.position.x * 100}
            top={handle.position.y * 100}
          />
        ))}
      </Handle.Boundary>
    </Primitive.Div>
  );
};
