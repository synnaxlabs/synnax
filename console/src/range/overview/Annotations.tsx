import { type annotation, type ontology, ranger, TimeRange } from "@synnaxlabs/client";
import {
  Align,
  Annotation,
  Button,
  Form,
  Header,
  Icon,
  List,
  Text,
} from "@synnaxlabs/pluto";
import { useMemo, useState } from "react";

export interface AnnotationListItemProps extends List.ItemProps<annotation.Key> {
  parent?: ontology.ID;
  initialEdit?: boolean;
}

export const AnnotationListItem = ({
  initialEdit = false,
  parent,
  ...rest
}: AnnotationListItemProps) => {
  const { itemKey } = rest;
  const initialValues = List.useItem<string, annotation.Annotation>(itemKey);
  const [edit, setEdit] = useState(initialEdit);
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
    sync: true,
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
  const { data, getItem, retrieve, subscribe } = Annotation.useList({
    initialParams: { parent },
  });
  const { fetchMore } = List.usePager({ retrieve });
  return (
    <Align.Space y>
      <Header.Header level="h4" bordered={false} borderShade={5}>
        <Header.Title shade={11} weight={450}>
          Annotations
        </Header.Title>
      </Header.Header>
      <List.Frame<annotation.Key, annotation.Annotation>
        data={data}
        getItem={getItem}
        onFetchMore={fetchMore}
        subscribe={subscribe}
      >
        <List.Items<annotation.Key>>
          {({ key, ...rest }) => (
            <AnnotationListItem key={key} parent={parent} {...rest} />
          )}
        </List.Items>
        <AnnotationListItem
          key="form"
          index={0}
          itemKey=""
          parent={parent}
          initialEdit
        />
      </List.Frame>
    </Align.Space>
  );
};
