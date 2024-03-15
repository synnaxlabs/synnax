// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useState } from "react";

import { zodResolver } from "@hookform/resolvers/zod";
import { label } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Synnax,
  Align,
  Color,
  Input,
  Button,
  useAsyncEffect,
  Text,
  Nav,
  Header,
} from "@synnaxlabs/pluto";
import { FormProvider, useForm } from "react-hook-form";
import { type z } from "zod";

import { CSS } from "@/css";
import { type Layout } from "@/layout";

import "@/label/Manage.css";

const MANAGE_LABELS_WINDOW_KEY = "manageLabels";

export const manageWindowLayout: Layout.LayoutState = {
  key: MANAGE_LABELS_WINDOW_KEY,
  type: MANAGE_LABELS_WINDOW_KEY,
  windowKey: MANAGE_LABELS_WINDOW_KEY,
  name: "Manage Labels",
  location: "window",
  window: {
    resizable: false,
    size: { height: 625, width: 625 },
    navTop: true,
    transparent: true,
  },
};

export const Manage = ({ onClose }: Layout.RendererProps): ReactElement => {
  const client = Synnax.use();

  const [labels, setLabels] = useState<label.Label[]>([]);
  const [editing, setEditing] = useState<number | null>(null);

  useAsyncEffect(async () => {
    if (client == null) return;

    const ct = await client.labels.openChangeTracker();

    ct.onChange((changes) => {
      const deleted = changes.filter((c) => c.variant === "delete").map((c) => c.key);
      const added = changes
        .filter((c) => c.variant === "set")
        .map((a) => a.value) as label.Label[];
      const addedKeys = added.map((v) => v.key);
      setLabels((l) => [
        ...l.filter((v) => !deleted.includes(v.key) && !addedKeys.includes(v.key)),
        ...added,
      ]);
    });

    const fetched = await client.labels.page(0, 500);
    setLabels(fetched);

    return () => {
      void ct.close();
    };
  }, [client]);

  const handleDelete = (key: label.Key): void => {
    if (client == null) return;
    client.labels.delete(key);
  };

  return (
    <Align.Space direction="y" grow className={CSS.B("manage-labels")}>
      <Header.Header level="h4">
        <Header.Title>Cluster Labels</Header.Title>
      </Header.Header>
      <Align.Space direction="y" grow align="center" style={{ padding: "2rem" }}>
        <Form />
        {labels
          .sort((a, b) => a.key.localeCompare(b.key))
          .map((l, i) => {
            if (editing === i)
              return (
                <Form isEdit defaultValues={l} onFinish={() => setEditing(null)} />
              );
            return (
              <ListItem
                key={l.key}
                label={l}
                onDelete={handleDelete}
                onEdit={() => setEditing(i)}
              />
            );
          })}
      </Align.Space>
      <Nav.Bar location="bottom" size={48}>
        <Nav.Bar.End style={{ padding: "2rem" }}>
          <Button.Button>Done</Button.Button>
        </Nav.Bar.End>
      </Nav.Bar>
    </Align.Space>
  );
};

interface FormProps {
  defaultValues?: z.input<typeof label.newLabelPayloadZ>;
  isEdit?: boolean;
  onFinish?: () => void;
}
const Form = ({
  isEdit = false,
  defaultValues = { name: "", color: Color.ZERO.setAlpha(1).hex },
  onFinish,
}: FormProps): ReactElement => {
  const client = Synnax.use();

  const methods = useForm({
    defaultValues,
    resolver: zodResolver(label.newLabelPayloadZ),
  });

  const onSubmit = (e: label.NewLabelPayload): void => {
    onFinish?.();
    if (client == null) return;
    client.labels.create(e);
    reset();
  };

  return (
    <Align.Space
      el="form"
      onSubmit={(e) => {
        e.preventDefault();
        void methods.handleSubmit(onSubmit)(e);
      }}
      direction="x"
    >
      <FormProvider {...methods}>
        <Input.HFItem showLabel={false} name="color">
          {({ value, onChange, ...props }) => (
            <Color.Swatch value={value} onChange={(c) => onChange(c.hex)} {...props} />
          )}
        </Input.HFItem>
        <Input.HFItem name="name" showLabel={false} grow>
          {(p) => <Input.Text placeholder="Name" {...p} />}
        </Input.HFItem>
        {!isEdit ? (
          <Button.Button
            variant="filled"
            type="submit"
            disabled={client == null}
            startIcon={<Icon.Add />}
          >
            New Label
          </Button.Button>
        ) : (
          <>
            <Button.Button variant="outlined" onClick={onFinish}>
              Cancel
            </Button.Button>
            <Button.Button variant="filled" type="submit">
              Save
            </Button.Button>
          </>
        )}
      </FormProvider>
    </Align.Space>
  );
};

interface ListItemProps {
  label: label.Label;
  onDelete: (key: label.Key) => void;
  onEdit: (key: label.Key) => void;
}

const ListItem = ({ label: l, onEdit, onDelete }: ListItemProps): ReactElement => {
  return (
    <Align.Space direction="x" justify="spaceBetween" className={CSS.B("list-item")}>
      <Text.WithIcon size="medium" level="p" startIcon={<Icon.Circle fill={l.color} />}>
        {l.name}
      </Text.WithIcon>
      <Align.Space
        className={CSS.BE("list-item", "actions")}
        direction="x"
        size="small"
      >
        <Button.Icon
          color="var(--pluto-gray-l8)"
          size="small"
          onClick={() => onEdit(l.key)}
        >
          <Icon.Edit />
        </Button.Icon>
        <Button.Icon
          color="var(--pluto-gray-l8)"
          size="small"
          onClick={() => onDelete(l.key)}
        >
          <Icon.Delete />
        </Button.Icon>
      </Align.Space>
    </Align.Space>
  );
};
