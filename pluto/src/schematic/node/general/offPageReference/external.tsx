// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { color } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Label } from "@/schematic/node/common/label";
import { OffPageReferenceForm } from "@/schematic/node/general/offPageReference/Form";
import { OffPageReference } from "@/schematic/node/general/offPageReference/Primitive";
import { Symbol } from "@/schematic/node/general/offPageReference/Symbol";
import { type Spec } from "@/schematic/node/spec";

export const defaultConfig = (): schematic.NodeConfigOffPageReference => ({
  variant: "off_page_reference",
  color: color.ZERO,
  orientation: "right",
  label: Label.defaultConfig("Off Page Reference"),
});

const Preview = ({
  label: _,
  ...rest
}: schematic.NodeConfigOffPageReference): ReactElement => (
  <OffPageReference label="Off Page" {...rest} orientation="right" />
);

export const spec: Spec<"off_page_reference", schematic.NodeConfigOffPageReference> = {
  key: "off_page_reference",
  name: "Off Page",
  Form: OffPageReferenceForm,
  Node: Symbol,
  Preview,
  defaultConfig,
  zIndex: 4,
};
