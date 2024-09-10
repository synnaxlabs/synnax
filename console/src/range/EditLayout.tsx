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
  Form,
  Icon as PIcon,
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

export const EDIT_LAYOUT_TYPE = "editRange";

const SAVE_TRIGGER: Triggers.Trigger = ["Control", "Enter"];

interface CreateEditLayoutProps extends Partial<Layout.State> {
  initial?: Partial<Args>;
}

export const createEditLayout = ({
  name,
  initial = {},
  window,
  ...rest
}: CreateEditLayoutProps): Layout.State => ({
  ...rest,
  key: EDIT_LAYOUT_TYPE,
  type: EDIT_LAYOUT_TYPE,
  windowKey: EDIT_LAYOUT_TYPE,
  icon: "Range",
  location: "modal",
  name: name ?? (initial.key != null ? "Range.Edit" : "Range.Create"),
  window: {
    resizable: false,
    size: { height: 370, width: 700 },
    navTop: true,
    ...window,
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
  const args = Layout.useSelectArgs<Args>(layoutKey);
  const range = useSelect(args.key);
  const client = Synnax.use();
  const isCreate = args.key == null;
  const isEdit = !isCreate && (range == null || range.persisted);
  const initialValues = useQuery<DefineRangeFormProps>({
    queryKey: ["range-edit", args],
    queryFn: async () => {
      if (isCreate)
        return {
          name: "",
          labels: [],
          timeRange: { start: now, end: now },
          parent: "",
          ...args,
        };
      if (range == null || range.persisted) {
        const key = args.key as string;
        if (client == null) throw new UnexpectedError("Client is not available");
        const rng = await client.ranges.retrieve(key);
        const parent = await rng.retrieveParent();
        const labels = await rng.labels();
        return {
          key: rng.key,
          name: rng.name,
          timeRange: rng.timeRange.numeric,
          labels: labels.map((l) => l.key),
          parent: parent?.key ?? "",
        };
      }
      if (range.variant !== "static") throw new UnexpectedError("Range is not static");
      return {
        key: range.key,
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
      isRemoteEdit={isEdit}
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

const EditLayoutForm = ({
  initialValues,
  isRemoteEdit,
  onClose,
}: EditLayoutFormProps): ReactElement => {
  const methods = Form.use({ values: deep.copy(initialValues), schema: formSchema });
  const dispatch = useDispatch();
  const client = Synnax.use();
  const addStatus = Status.useAggregator();
  const isCreate = initialValues.key == null;

  const { mutate, isPending } = useMutation({
    mutationFn: async (persist: boolean) => {
      if (!methods.validate()) return;
      const values = methods.value();
      const { timeRange: tr, parent } = methods.value();
      const timeRange = new TimeRange(tr);
      const name = values.name.trim();
      const key = isCreate ? uuidv4() : (initialValues.key as string);
      const persisted = persist || isRemoteEdit;
      const parentID = primitiveIsZero(parent)
        ? undefined
        : new ontology.ID({ key: parent as string, type: "range" });
      const otgID = new ontology.ID({ key, type: "range" });
      if (persisted && client != null) {
        await client.ranges.create({ key, name, timeRange }, { parent: parentID });
        await client.labels.label(otgID, values.labels, { replace: true });
      }
      dispatch(
        add({
          ranges: [
            { variant: "static", name, timeRange: timeRange.numeric, key, persisted },
          ],
        }),
      );
      onClose();
    },
    onError: (e) => addStatus({ message: e.message, variant: "error" }),
  });

  // Makes sure the user doesn't have the option to select the range itself as a parent
  const recursiveParentFilter = useCallback(
    (data: ranger.Range[]) => data.filter((r) => r.key !== initialValues.key),
    [initialValues.key],
  );

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
                  filter={recursiveParentFilter}
                  entryRenderKey={(e: ranger.Range) => (
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
                  onChange={(v: string) => onChange(v ?? "")}
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
      <Layout.BottomNavBar>
        <Nav.Bar.Start size="small">
          <Triggers.Text shade={7} level="small" trigger={SAVE_TRIGGER} />
          <Text.Text shade={7} level="small">
            To Save
          </Text.Text>
        </Nav.Bar.Start>
        <Nav.Bar.End>
          <Button.Button
            onClick={() => mutate(true)}
            disabled={isPending}
            triggers={[SAVE_TRIGGER]}
          >
            Save
          </Button.Button>
        </Nav.Bar.End>
      </Layout.BottomNavBar>
    </Align.Space>
  );
};
