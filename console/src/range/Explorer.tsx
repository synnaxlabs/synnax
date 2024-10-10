// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/range/EditLayout.css";

import {
  ontology,
  ranger,
  TimeRange,
  TimeStamp,
  UnexpectedError,
} from "@synnaxlabs/client";
import { Icon, Logo } from "@synnaxlabs/media";
import {
  Align,
  Button,
  componentRenderProp,
  Form,
  Icon as PIcon,
  List,
  Nav,
  Ranger,
  Status,
  Synnax,
  Text,
  Triggers,
} from "@synnaxlabs/pluto";
import { Input } from "@synnaxlabs/pluto/input";
import { deep, primitiveIsZero } from "@synnaxlabs/x";
import { useMutation, useQuery } from "@tanstack/react-query";
import { type ReactElement, useCallback, useRef } from "react";
import { useDispatch } from "react-redux";
import { v4 as uuidv4 } from "uuid";
import { z } from "zod";

import { CSS } from "@/css";
import { Label } from "@/label";
import { Layout } from "@/layout";
import { overviewLayout } from "@/range/external";
import { useSelect } from "@/range/selectors";
import { add } from "@/range/slice";

const formSchema = z.object({
  key: z.string().optional(),
  name: z.string().min(1, "Name must not be empty"),
  timeRange: z.object({ start: z.number(), end: z.number() }),
  labels: z.string().array(),
  parent: z.string().optional(),
});

type Args = Partial<z.infer<typeof formSchema>>;

export const EXPLORER_LAYOUT_TYPE = "explore_rates";

const SAVE_TRIGGER: Triggers.Trigger = ["Control", "Enter"];

interface CreateEditLayoutProps extends Partial<Layout.State> {}

export const createExplorerLayout = ({
  name,
  window,
  ...rest
}: CreateEditLayoutProps): Layout.State => ({
  ...rest,
  key: EXPLORER_LAYOUT_TYPE,
  type: EXPLORER_LAYOUT_TYPE,
  windowKey: EXPLORER_LAYOUT_TYPE,
  icon: "Range",
  location: "mosaic",
  name: "Explore Ranges",
});

type DefineRangeFormProps = z.infer<typeof formSchema>;

const parentRangeIcon = (
  <PIcon.Icon bottomRight={<Icon.Arrow.Up />}>
    <Icon.Range />
  </PIcon.Icon>
);

export const RangeListItem = (props: List.ItemProps<string, ranger.Payload>) => {
  const { entry } = props;
  const placer = Layout.usePlacer();
  return (
    <List.ItemFrame
      onClick={() => placer({ ...overviewLayout, name: entry.name, key: entry.key })}
      direction="x"
      size={0.5}
      justify="spaceBetween"
      align="center"
      style={{ padding: "1.5rem" }}
      {...props}
    >
      <Text.WithIcon
        startIcon={<Icon.Range style={{ color: "var(--pluto-gray-l9)" }} />}
        level="p"
        weight={450}
        shade={9}
        size="small"
      >
        {entry.name}
      </Text.WithIcon>
      <Ranger.TimeRangeChip level="p" timeRange={entry.timeRange} showSpan />
    </List.ItemFrame>
  );
};

const rangeListItem = componentRenderProp(RangeListItem);

export const Explore: Layout.Renderer = (props): ReactElement => {
  const { layoutKey } = props;
  const client = Synnax.use();

  return (
    <List.List>
      <List.Core></List.Core>
    </List.List>
  );
};
