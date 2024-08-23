import { ontology } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Align,
  componentRenderProp,
  List,
  Synnax,
  Text,
  useAsyncEffect,
} from "@synnaxlabs/pluto";
import { FC, ReactElement, useState } from "react";

import { Task } from "@/hardware/task";
import { Layout } from "@/layout";
import { create } from "@/schematic/external";

interface SnapshotService {
  icon: ReactElement;
  onClick: (res: ontology.Resource, placer: Layout.Placer) => void;
}

const SNAPSHOTS: Record<"schematic" | "task", SnapshotService> = {
  schematic: {
    icon: <Icon.Schematic />,
    onClick: (res, placer) => {
      placer(create({ key: res.key }));
    },
  },
  task: {
    icon: <Icon.Task />,
    onClick: (res, placer) => {
      placer(Task.createTaskLayout(res.id.key, res.data?.type as string));
    },
  },
};

const SnapshotsListItem = (props: List.ItemProps<string, ontology.Resource>) => {
  const { entry } = props;
  const { id, name } = entry;
  const svc = SNAPSHOTS[id.type];
  const placeLayout = Layout.usePlacer();
  console.log(svc);
  return (
    <List.ItemFrame
      style={{ padding: "1.5rem" }}
      size={0.5}
      {...props}
      onSelect={() => svc.onClick(entry, placeLayout)}
    >
      <Text.WithIcon startIcon={svc.icon} level="p" weight={450} shade={9}>
        {name}
      </Text.WithIcon>
    </List.ItemFrame>
  );
};

const snapshotsListItem = componentRenderProp(SnapshotsListItem);

export interface SnapshotsProps {
  rangeKey: string;
}

export const Snapshots: FC<SnapshotsProps> = ({ rangeKey }) => {
  const client = Synnax.use();
  const [snapshots, setSnapshots] = useState<ontology.Resource[]>([]);

  useAsyncEffect(async () => {
    if (client == null) return;
    const otgID = new ontology.ID({ type: "range", key: rangeKey });
    const children = await client.ontology.retrieveChildren(otgID);
    const relevant = children.filter((child) => child.data?.snapshot === true);
    setSnapshots(relevant);
    const tracker = await client.ontology.openDependentTracker(
      otgID,
      relevant,
      "parent",
      "to",
    );
    tracker.onChange((snapshots) => {
      const relevant = snapshots.filter((child) => child.data?.snapshot === true);
      setSnapshots(relevant);
    });
  }, [client, rangeKey]);

  return (
    <Align.Space direction="y">
      <Text.Text level="h4" shade={8} weight={500}>
        Snapshots
      </Text.Text>
      <List.List data={snapshots}>
        <List.Core empty>{snapshotsListItem}</List.Core>
      </List.List>
    </Align.Space>
  );
};
