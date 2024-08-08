import { ontology, ranger } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Button,
  componentRenderProp,
  List,
  Observe,
  Ranger,
  Synnax,
  Text,
  useAsyncEffect,
} from "@synnaxlabs/pluto";
import { use } from "@synnaxlabs/pluto/dist/src/dialog/use.js";
import { useQuery } from "@tanstack/react-query";
import { FC, useState } from "react";

import { Layout } from "@/layout";
import { createEditLayout, overviewLayout } from "@/range/external";

export const SubRangeListItem = (props: List.ItemProps<string, ranger.Payload>) => {
  const { entry } = props;
  const placer = Layout.usePlacer();
  return (
    <List.ItemFrame
      onClick={() => placer({ ...overviewLayout, name: entry.name, key: entry.key })}
      direction="y"
      size={0.5}
      style={{ padding: "1.5rem" }}
      {...props}
    >
      <Text.WithIcon
        startIcon={
          <Icon.Range
            style={{ transform: "scale(0.9)", color: "var(--pluto-gray-l9)" }}
          />
        }
        level="p"
        weight={450}
        shade={9}
        size="small"
      >
        {entry.name}{" "}
      </Text.WithIcon>
      <Ranger.TimeRangeChip level="small" timeRange={entry.timeRange} />
    </List.ItemFrame>
  );
};

const subRangeListItem = componentRenderProp(SubRangeListItem);

export const SubRanges: FC<{ rng: ranger.Range }> = ({ rng }) => {
  const placer = Layout.usePlacer();
  const [subRanges, setSubRanges] = useState<ranger.Range[]>([]);

  useAsyncEffect(async () => {
    const subRanges = await rng.retrieveChildren();
    setSubRanges(subRanges);
    const tracker = await rng.openSubRangeTracker();
    tracker.onChange((ranges) => setSubRanges(ranges));
    return async () => await tracker.close();
  }, [rng.key]);

  return (
    <Align.Space direction="y">
      <Text.Text level="h4" shade={8} weight={500}>
        Sub Ranges
      </Text.Text>
      <List.List data={subRanges}>
        <List.Core empty>{subRangeListItem}</List.Core>
      </List.List>
      <Button.Button
        size="small"
        shade={8}
        weight={500}
        startIcon={<Icon.Add />}
        variant="text"
        onClick={() => {
          placer(createEditLayout(undefined, { parent: rng.key }));
        }}
      >
        Add Sub Range
      </Button.Button>
    </Align.Space>
  );
};
