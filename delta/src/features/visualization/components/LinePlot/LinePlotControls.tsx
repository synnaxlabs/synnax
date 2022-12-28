import { useState } from "react";

import { Synnax } from "@synnaxlabs/client";
import type { ChannelPayload } from "@synnaxlabs/client";
import { Select, Space } from "@synnaxlabs/pluto";

import { LinePlotVisualization, SugaredLinePlotVisualization } from "../../types";

import { useSelectRanges } from "@/features/workspace";
import { useAsyncEffect } from "@/hooks";

export interface LinePlotControlsProps {
  visualization: SugaredLinePlotVisualization;
  onChange: (vis: LinePlotVisualization) => void;
  client: Synnax;
}

export const LinePlotControls = ({
  visualization,
  onChange,
  client,
}: LinePlotControlsProps): JSX.Element => {
  return <></>;
};
