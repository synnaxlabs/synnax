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
  Synnax as Client,
  TimeRange,
  TimeStamp,
  UnexpectedError,
} from "@synnaxlabs/client";
import { Icon, Logo } from "@synnaxlabs/media";
import {
  Align,
  Button,
  Form,
  Icon as PIcon,
  Label,
  Nav,
  Ranger,
  Status,
  Synnax,
  Text,
  Triggers,
} from "@synnaxlabs/pluto";
import { Input } from "@synnaxlabs/pluto/input";
import { deep } from "@synnaxlabs/x";
import { useMutation, useQuery } from "@tanstack/react-query";
import { type ReactElement, useRef } from "react";
import { useDispatch } from "react-redux";
import { v4 as uuidv4 } from "uuid";
import { z } from "zod";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { type TimeRange as TimeRangeT } from "@/range/migrations";
import { useSelect } from "@/range/selectors";
import { add, StaticRange } from "@/range/slice";

const formSchema = z.object({
  name: z.string().min(1, "Name must not be empty"),
  timeRange: z.object({
    start: z.number().int(),
    end: z.number().int(),
  }),
  labels: z.string().array(),
  parent: z.string().optional(),
});

type FormSchema = z.infer<typeof formSchema>;

export const EDIT_LAYOUT_TYPE = "editRange";

const SAVE_TRIGGER: Triggers.Trigger = ["Control", "Enter"];

export const createEditLayout = (
  name: string = "Range.Create",
  initial?: Partial<FormSchema>,
): Layout.State => ({
  key: EDIT_LAYOUT_TYPE,
  type: EDIT_LAYOUT_TYPE,
  windowKey: EDIT_LAYOUT_TYPE,
  name,
  icon: "Range",
  location: "modal",
  window: {
    resizable: false,
    size: { height: 370, width: 700 },
    navTop: true,
  },
  args: initial,
});

type DefineRangeFormProps = z.infer<typeof formSchema>;

const parentRangeIcon = (
  <PIcon.Icon bottomRight={<Icon.Arrow.Up />}>
    <Icon.Range />
  </PIcon.Icon>
);

export const Edit = (props: Layout.RendererProps): ReactElement => {
  const { layoutKey } = props;
  const now = useRef(Number(TimeStamp.now().valueOf())).current;
  const range = useSelect(layoutKey);
  const args = Layout.useSelectArgs<Partial<FormSchema>>(layoutKey);
  const client = Synnax.use();
  const isCreate = layoutKey === EDIT_LAYOUT_TYPE;
  const isRemoteEdit = !isCreate && (range == null || range.persisted);
  const initialValues = useQuery<DefineRangeFormProps>({
    queryKey: ["range", layoutKey],
    queryFn: async () => {
      if (isCreate) {
        return {
          name: "",
          labels: [],
          timeRange: {
            start: now,
            end: now,
          },
          parent: "",
          ...args,
        };
      }
      if (range == null || range.persisted) {
        if (client == null) throw new UnexpectedError("Client is not available");
        const rng = await client.ranges.retrieve(layoutKey);
        const parent = await client.ontology.retrieveParents(
          new ontology.ID({ key: layoutKey, type: "range" }),
        );
        const labels = await client.labels.retrieveFor(
          new ontology.ID({ key: layoutKey, type: "range" }),
        );
        return {
          name: rng.name,
          timeRange: {
            start: Number(rng.timeRange.start.valueOf()),
            end: Number(rng.timeRange.end.valueOf()),
          },
          labels: labels.map((l) => l.key),
          parent: parent.length > 0 ? parent[0].id.key : "",
        };
      }
      if (range.variant !== "static") throw new UnexpectedError("Range is not static");
      return {
        name: range.name,
        timeRange: range.timeRange,
        labels: [],
        parent: "",
      };
    },
  });
  if (initialValues.isPending) return <Logo.Watermark variant="loader" />;
  if (initialValues.isError) throw initialValues.error;
  return (
    <EditLayoutForm
      isRemoteEdit={isRemoteEdit}
      initialValues={initialValues.data}
      {...props}
    />
  );
};

interface EditLayoutFormProps extends Layout.RendererProps {
  initialValues: DefineRangeFormProps;
  isRemoteEdit: boolean;
  onClose: () => void;
}

export const updateLabels = async (
  client: Client,
  key: string,
  prevLabels: string[],
  labels: string[],
): Promise<void> => {
  const removed = prevLabels.filter((l) => !labels.includes(l));
  await client.labels.label(new ontology.ID({ key, type: "range" }), labels);
  await client.labels.removeLabels(new ontology.ID({ key, type: "range" }), removed);
};

const EditLayoutForm = ({
  layoutKey,
  initialValues,
  isRemoteEdit,
  onClose,
}: EditLayoutFormProps): ReactElement => {
  const methods = Form.use({ values: deep.copy(initialValues), schema: formSchema });
  const dispatch = useDispatch();
  const client = Synnax.use();
  const isCreate = layoutKey === EDIT_LAYOUT_TYPE;
  const addStatus = Status.useAggregator();

  const { mutate, isPending } = useMutation({
    mutationFn: async (persist: boolean) => {
      if (!methods.validate()) return;
      const values = methods.value();
      const { timeRange } = methods.value();

      const startTS = new TimeStamp(timeRange.start, "UTC");
      const endTS = new TimeStamp(timeRange.end, "UTC");
      const name = values.name.trim();
      const key = isCreate ? uuidv4() : layoutKey;
      const persisted = persist || isRemoteEdit;
      const tr = new TimeRange(startTS, endTS);
      if (persisted && client != null) {
        const parent = values.parent;
        await client.ranges.create(
          { key, name, timeRange: tr },
          {
            parent:
              parent != null && parent !== ""
                ? new ontology.ID({ key: parent, type: "range" })
                : undefined,
          },
        );
        await updateLabels(client, key, initialValues.labels, values.labels);
        if (parent != null && parent !== "") {
          if (!isCreate) {
            if (initialValues.parent != null)
              await client?.ontology.moveChildren(
                new ontology.ID({ key: initialValues.parent, type: "range" }),
                new ontology.ID({ key: parent, type: "range" }),
                new ontology.ID({ key: layoutKey, type: "range" }),
              );
            else
              await client?.ontology.addChildren(
                new ontology.ID({ key: parent, type: "range" }),
                new ontology.ID({ key: layoutKey, type: "range" }),
              );
          }
        }
      }
      dispatch(
        add({
          ranges: [
            {
              variant: "static",
              name,
              timeRange: {
                start: Number(startTS.valueOf()),
                end: Number(endTS.valueOf()),
              },
              key,
              persisted,
            },
          ],
        }),
      );
      onClose();
    },
    onError: (e) => addStatus({ message: e.message, variant: "error" }),
  });

  return (
    <Align.Space className={CSS.B("range-edit-layout")} grow empty>
      <Align.Space
        className="console-form"
        justify="center"
        style={{ padding: "1rem 3rem" }}
        grow
      >
        <Form.Form {...methods}>
          <Form.Field<string> path="name">
            {(p) => (
              <Input.Text
                autoFocus
                level="h2"
                variant="natural"
                placeholder="Range Name"
                {...p}
              />
            )}
          </Form.Field>
          <Align.Space direction="x" size="large">
            <Form.Field<number> path="timeRange.start" label="From">
              {(p) => <Input.DateTime level="h4" variant="natural" {...p} />}
            </Form.Field>
            <Text.WithIcon level="h4" startIcon={<Icon.Arrow.Right />} />
            <Form.Field<number> path="timeRange.end" label="To">
              {(p) => <Input.DateTime level="h4" variant="natural" {...p} />}
            </Form.Field>
          </Align.Space>
          <Align.Space direction="x">
            <Form.Field<string>
              path="parent"
              visible={isCreate || isRemoteEdit}
              padHelpText={false}
            >
              {({ onChange, ...p }) => (
                <Ranger.SelectSingle
                  dropdownVariant="modal"
                  style={{ width: "fit-content" }}
                  zIndex={100}
                  entryRenderKey={(e) => (
                    <Text.WithIcon
                      level="p"
                      shade={9}
                      startIcon={parentRangeIcon}
                      size="small"
                    >
                      {e.name}
                    </Text.WithIcon>
                  )}
                  inputPlaceholder={"Search Ranges"}
                  triggerTooltip={"Select Parent Range"}
                  placeholder={
                    <Text.WithIcon
                      level="p"
                      shade={7}
                      startIcon={parentRangeIcon}
                      size="small"
                    >
                      Parent Range
                    </Text.WithIcon>
                  }
                  onChange={(v) => {
                    onChange(v ?? "");
                  }}
                  {...p}
                />
              )}
            </Form.Field>
            <Form.Field<string> path="labels" required={false}>
              {(p) => (
                <Label.SelectMultiple
                  searcher={client?.labels}
                  entryRenderKey="name"
                  dropdownVariant="floating"
                  zIndex={100}
                  location="bottom"
                  {...p}
                />
              )}
            </Form.Field>
          </Align.Space>
        </Form.Form>
      </Align.Space>
      <Nav.Bar location="bottom" size={48}>
        <Nav.Bar.Start style={{ paddingLeft: "2rem" }} size="small">
          <Triggers.Text shade={7} level="small" trigger={SAVE_TRIGGER} />
          <Text.Text shade={7} level="small">
            To Save
          </Text.Text>
        </Nav.Bar.Start>
        <Nav.Bar.End style={{ padding: "1rem" }}>
          <Button.Button variant="outlined" onClick={() => mutate(false)}>
            Save {!isRemoteEdit && "Locally"}
          </Button.Button>
          {(isCreate || !isRemoteEdit) && (
            <Button.Button
              onClick={() => mutate(true)}
              disabled={client == null || isPending}
              loading={isPending}
              triggers={[SAVE_TRIGGER]}
            >
              Save to Synnax
            </Button.Button>
          )}
        </Nav.Bar.End>
      </Nav.Bar>
    </Align.Space>
  );
};
