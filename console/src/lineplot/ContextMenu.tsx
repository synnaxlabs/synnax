// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Icon } from "@synnaxlabs/media";
import { Menu as Core } from "@synnaxlabs/pluto";
import { TimeRange, box, scale } from "@synnaxlabs/x";

import { Menu } from "@/components";
import { useSelectAxisBounds, useSelectSelection } from "@/lineplot/selectors";
import { download } from "@/lineplot/download";
