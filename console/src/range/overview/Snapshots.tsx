import { ontology } from "@synnaxlabs/client";
import {
  Align,
  componentRenderProp,
  List,
  Synnax,
  Text,
  useAsyncEffect,
} from "@synnaxlabs/pluto";
import { FC, useState } from "react";

const SnapshotsListItem = (props: List.ItemProps<string, ontology.Resource>) => {
  return <p>{JSON.stringify(props)}</p>;
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
  });

  return (
    <Align.Space direction="y">
      <Text.Text level="h4">Snapshots</Text.Text>
      <List.List data={snapshots}>
        <List.Core empty>{snapshotsListItem}</List.Core>
      </List.List>
    </Align.Space>
  );
};
