// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";
import type { ReactElement } from "react";

import { Color } from "@/color";
import {
  colorFallbacks,
  OffPageReferenceForm,
} from "@/schematic/node/general/offPageReference/Form";
import { OffPageReference } from "@/schematic/node/general/offPageReference/Primitive";
import { Symbol } from "@/schematic/node/general/offPageReference/Symbol";
import type { Spec } from "@/schematic/node/spec";

const Preview = ({
  label: _,
  fillColor,
  ...rest
}: schematic.OffPageReferenceNodeConfig): ReactElement => {
  const resolve = Color.useResolve();
  return (
    <OffPageReference
      label="Off page"
      {...rest}
      fillColor={resolve(fillColor)}
      orientation="right"
    />
  );
};

export const spec: Spec<"off_page_reference", schematic.OffPageReferenceNodeConfig> = {
  key: "off_page_reference",
  name: "Off page",
  Form: OffPageReferenceForm,
  Node: Symbol,
  Preview,
  zIndex: 4,
  colorFallbacks,
};
