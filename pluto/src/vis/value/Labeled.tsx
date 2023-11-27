// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useState } from "react";

import { box, direction, type xy } from "@synnaxlabs/x";

import { Align } from "@/align";
import { Color } from "@/color";
import { CSS } from "@/css";
import { useResize } from "@/hooks";
import { Text } from "@/text";
import { Theming } from "@/theming";
import { type UseTypographyReturn } from "@/theming/font";
import { use, type UseProps } from "@/vis/value/Core";

import "@/vis/value/Labeled.css";
