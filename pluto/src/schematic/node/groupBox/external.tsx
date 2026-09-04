// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { type Config, VARIANT } from "@/schematic/node/groupBox/config";
import { GroupBoxForm } from "@/schematic/node/groupBox/Form";
import { Symbol } from "@/schematic/node/groupBox/Symbol";
import { type Spec } from "@/schematic/node/spec";

export * from "@/schematic/node/groupBox/config";

const NAME = "Group";

export const defaultConfig = (): Config => ({
  variant: VARIANT,
  members: [],
  dimensions: { width: 100, height: 100 },
});

const Preview = (): ReactElement => <div />;

export const spec: Spec<typeof VARIANT, Config> = {
  key: VARIANT,
  name: NAME,
  Form: GroupBoxForm,
  Node: Symbol,
  Preview,
  defaultConfig,
  zIndex: -1,
};
