import "@/annotation/list/List.css";

import { type annotation, type ontology, type TimeStamp } from "@synnaxlabs/client";
import { Annotation, List as PList } from "@synnaxlabs/pluto";

import { Item } from "@/annotation/list/Item";
import { CSS } from "@/css";

export interface AnnotationListProps {
  parent: ontology.ID;
  parentStart?: TimeStamp;
}

export const List = ({ parent, parentStart }: AnnotationListProps) => {
  const { data, getItem, retrieve, subscribe } = Annotation.useList({
    initialQuery: { parent },
  });
  const { fetchMore } = PList.usePager({ retrieve });

  return (
    <PList.Frame<annotation.Key, annotation.Annotation>
      data={data}
      getItem={getItem}
      onFetchMore={fetchMore}
      subscribe={subscribe}
      virtual={false}
    >
      <PList.Items<annotation.Key>
        gap="medium"
        className={CSS.BE("annotation", "list")}
      >
        {({ key, ...rest }) => (
          <Item key={key} parent={parent} parentStart={parentStart} {...rest} />
        )}
      </PList.Items>
      <Item key="form" index={0} itemKey="" parent={parent} isCreate />
    </PList.Frame>
  );
};
