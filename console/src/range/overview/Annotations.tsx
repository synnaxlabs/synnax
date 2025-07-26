import {
  annotation,
  type ontology,
  ranger,
  TimeRange,
  type TimeStamp,
} from "@synnaxlabs/client";
import {
  Align,
  Annotation,
  Button,
  Form,
  Header,
  Icon,
  List,
  Ranger,
  Text,
  User as PUser,
} from "@synnaxlabs/pluto";
import { useMemo, useState } from "react";

export interface AnnotationListItemProps extends List.ItemProps<annotation.Key> {
  parent?: ontology.ID;
  isCreate?: boolean;
  parentStart?: TimeStamp;
}

export const AnnotationListItem = ({
  parent,
  parentStart,
  isCreate,
  ...rest
}: AnnotationListItemProps) => {
  const { itemKey } = rest;
  const initialValues = List.useItem<string, annotation.Annotation>(itemKey);
  const [edit, setEdit] = useState(isCreate);
  const values = useMemo(
    () => ({
      key: itemKey.length > 0 ? itemKey : undefined,
      message: initialValues?.message ?? "",
      timeRange: initialValues?.timeRange.numeric ?? TimeRange.ZERO.numeric,
    }),
    [initialValues],
  );
  const { form, save } = Annotation.useForm({
    params: { parent },
    initialValues: values,
    sync: !isCreate,
    afterSave: ({ form }) => {
      if (isCreate) form.reset();
    },
  });
  const { data: creator } = PUser.retrieveCreator.useDirect({
    params: { id: annotation.ontologyID(itemKey) },
  });

  return (
    <List.Item
      {...rest}
      bordered
      background={2}
      rounded={1}
      variant="outlined"
      borderShade={6}
      y
      style={{ maxHeight: "100%", height: "fit-content", minHeight: "fit-content" }}
    >
      <Align.Space x grow justify="spaceBetween">
        <Align.Space x>
          <Text.Text level="h5" shade={9} weight={450}>
            {creator?.username}
          </Text.Text>
        </Align.Space>
        {initialValues?.timeRange && (
          <Ranger.TimeRangeChip
            level="small"
            timeRange={initialValues.timeRange}
            collapseZero
            offsetFrom={parentStart}
          />
        )}
      </Align.Space>
      <Form.Form<typeof Annotation.formSchema> {...form}>
        {edit ? (
          <Form.TextAreaField
            path="message"
            showLabel={false}
            inputProps={{ placeholder: "Leave a comment...", level: "h5" }}
          />
        ) : (
          <Text.Text level="h5" shade={11} weight={450}>
            {initialValues?.message}
          </Text.Text>
        )}
      </Form.Form>
      {edit && (
        <Align.Space x grow justify="end">
          <Button.Icon variant="outlined" shade={2} onClick={() => save()}>
            <Icon.Arrow.Up />
          </Button.Icon>
        </Align.Space>
      )}
    </List.Item>
  );
};

export interface AnnotationsProps {
  rangeKey: string;
}

export const Annotations = ({ rangeKey }: AnnotationsProps) => {
  const parent = useMemo(() => ranger.ontologyID(rangeKey), [rangeKey]);
  const range = Ranger.useRetrieve({ params: { key: rangeKey } });
  const { data, getItem, retrieve, subscribe } = Annotation.useList({
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
        <List.Items<annotation.Key> size="medium">
          {({ key, ...rest }) => (
            <AnnotationListItem
              key={key}
              parent={parent}
              parentStart={range?.data?.timeRange.start}
              {...rest}
            />
          )}
        </List.Items>
        <AnnotationListItem key="form" index={0} itemKey="" parent={parent} isCreate />
      </List.Frame>
    </Align.Space>
  );
};
