// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Theming } from "@synnaxlabs/charon";
import { type ReactElement } from "react";

import { Label } from "@/schematic/node/common/label";
import { type Config, VARIANT } from "@/schematic/node/general/offPageReference/config";
import { OffPageReferenceForm } from "@/schematic/node/general/offPageReference/Form";
import { Primitive } from "@/schematic/node/general/offPageReference/Primitive";
import { Symbol } from "@/schematic/node/general/offPageReference/Symbol";
import { type Spec } from "@/schematic/node/spec";

export * from "@/schematic/node/general/offPageReference/config";

export const defaultConfig = (t: Theming.Theme): Config => ({
  variant: VARIANT,
  color: t.colors.gray.l11,
  orientation: "right",
  label: Label.defaultConfig("Off Page Reference"),
});

const Preview = ({ label: _, ...rest }: Config): ReactElement => (
  <Primitive label="Off Page" {...rest} orientation="right" />
);

export const spec: Spec<typeof VARIANT, Config> = {
  key: VARIANT,
  name: "Off Page",
  Form: OffPageReferenceForm,
  Node: Symbol,
  Preview,
  defaultConfig,
  zIndex: 4,
};
