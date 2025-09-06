// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ChannelServices } from "@/channel/services";
import { ClusterServices } from "@/cluster/services";
import { Docs } from "@/docs";
import { Hardware } from "@/hardware";
import { LabelServices } from "@/label/services";
import { Layout } from "@/layout";
import { LinePlotServices } from "@/lineplot/services";
import { LogServices } from "@/log/services";
import { type Palette } from "@/palette";
import { Persist } from "@/persist";
import { RangeServices } from "@/range/services";
import { SchematicServices } from "@/schematic/services";
import { TableServices } from "@/table/services";
import { UserServices } from "@/user/services";
import { WorkspaceServices } from "@/workspace/services";

export const COMMANDS: Palette.Command[] = [
  ...ChannelServices.COMMANDS,
  ...ClusterServices.COMMANDS,
  ...Docs.COMMANDS,
  ...Hardware.COMMANDS,
  ...LabelServices.COMMANDS,
  ...Layout.COMMANDS,
  ...LinePlotServices.COMMANDS,
  ...LogServices.COMMANDS,
  ...Persist.COMMANDS,
  ...RangeServices.COMMANDS,
  ...SchematicServices.COMMANDS,
  ...TableServices.COMMANDS,
  ...UserServices.COMMANDS,
  ...WorkspaceServices.COMMANDS,
];
