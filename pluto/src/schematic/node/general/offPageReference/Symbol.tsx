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
import { Grid } from "@/schematic/node/common/grid";
import {
  OffPageReference,
  offPageReferenceTooltip,
} from "@/schematic/node/general/offPageReference/Primitive";
import type { NodeProps } from "@/schematic/node/spec";

export const Symbol = ({
  onConfigChange,
  config: {
    label: labelConfig,
    orientation,
    fillColor,
    page,
    doubleClickNavigationDisabled,
  },
}: NodeProps<schematic.OffPageReferenceNodeConfig>): ReactElement => {
  const resolve = Color.useResolve();
  return (
    <OffPageReference
      className={Grid.DRAG_HANDLE_CLASS}
      onLabelChange={(label) => onConfigChange({ label: { ...labelConfig, label } })}
      label={labelConfig.label}
      level={labelConfig.level}
      orientation={orientation}
      fillColor={resolve(fillColor)}
      linked={page.key.length > 0}
      pageType={page.type}
      title={offPageReferenceTooltip(page, doubleClickNavigationDisabled)}
    />
  );
};
