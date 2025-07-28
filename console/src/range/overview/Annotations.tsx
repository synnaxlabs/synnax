import { type annotation, ranger } from "@synnaxlabs/client";
import {
  Align,
  Annotation as PAnnotation,
  Header,
  List,
  Ranger,
} from "@synnaxlabs/pluto";
import { useMemo } from "react";

import { Annotation } from "@/annotation";

export interface AnnotationsProps {
  rangeKey: string;
}

export const Annotations = ({ rangeKey }: AnnotationsProps) => {
  const parent = useMemo(() => ranger.ontologyID(rangeKey), [rangeKey]);
  const range = Ranger.useRetrieve({ params: { key: rangeKey } });
  const { data, getItem, retrieve, subscribe } = PAnnotation.useList({
    initialParams: { parent },
  });
  const { fetchMore } = List.usePager({ retrieve });

  return (
    <Align.Space y>
      <Header.Header level="h4" bordered={false} borderShade={5} padded>
        <Header.Title shade={11} weight={450}>
          Annotations
        </Header.Title>
      </Header.Header>
      <List.Frame<annotation.Key, annotation.Annotation>
        data={data}
        getItem={getItem}
        onFetchMore={fetchMore}
        subscribe={subscribe}
        virtual={false}
      >
        <List.Items<annotation.Key> gap="medium">
          {({ key, ...rest }) => (
            <Annotation.ListItem
              key={key}
              parent={parent}
              parentStart={range?.data?.timeRange.start}
              {...rest}
            />
          )}
        </List.Items>
        <Annotation.ListItem key="form" index={0} itemKey="" parent={parent} isCreate />
      </List.Frame>
    </Align.Space>
  );
};
