// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useRef, type ReactElement, useState, useMemo } from "react";

import { zodResolver } from "@hookform/resolvers/zod";
import { TimeRange, TimeStamp } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Align, Button, Nav, Synnax, Text, useAsyncEffect } from "@synnaxlabs/pluto";
import { Input } from "@synnaxlabs/pluto/input";
import { FormProvider, useForm } from "react-hook-form";
import { useDispatch } from "react-redux";
import { v4 as uuidv4 } from "uuid";
import { z } from "zod";

import { CSS } from "@/css";
import { type Layout } from "@/layout";
import { useSelect } from "@/range/selectors";
import { add } from "@/range/slice";

import "@/range/EditLayout.css";

const formSchema = z.object({
  name: z.string().min(1, "Name must not be empty"),
  start: z.number().int(),
  end: z.number().int(),
  labels: z.string().array(),
});

const RANGE_WINDOW_KEY = "defineRange";

export const editLayout = (name: string = "Create Range"): Layout.LayoutState => ({
  key: RANGE_WINDOW_KEY,
  type: RANGE_WINDOW_KEY,
  windowKey: RANGE_WINDOW_KEY,
  name,
  location: "window",
  window: {
    resizable: false,
    size: { height: 280, width: 700 },
    navTop: true,
    transparent: true,
  },
});

type DefineRangeFormProps = z.infer<typeof formSchema>;

export const EditLayout = ({
  layoutKey,
  onClose,
}: Layout.RendererProps): ReactElement => {
  const now = useMemo(() => TimeStamp.now().valueOf(), []);
  const range = useSelect(layoutKey);
  const [loading, setLoading] = useState(false);
  const client = Synnax.use();
  let defaultValues: DefineRangeFormProps = {
    name: "",
    start: now,
    end: now,
    labels: [],
  };
  const isCreate = layoutKey === RANGE_WINDOW_KEY;
  const isEdit = client != null && !isCreate && range != null;
  const isRemoteEdit = isEdit && range?.variant === "static" && range.persisted;

  if (range != null && range.variant === "static")
    defaultValues = {
      name: range.name,
      start: range.timeRange.start,
      end: range.timeRange.end,
      labels: [],
    };

  const methods = useForm({
    defaultValues,
    resolver: zodResolver(formSchema),
  });

  useAsyncEffect(async () => {
    if (!isRemoteEdit) return;
    const rng = await client.ranges.retrieve(layoutKey);
    methods.reset({
      name: rng.name,
      start: rng.timeRange.start.valueOf(),
      end: rng.timeRange.start.valueOf(),
      labels: [],
    });
  }, [isEdit]);

  const dispatch = useDispatch();
  const savePermanently = useRef(false);

  const onSubmit = async ({
    name,
    start,
    end,
    labels,
  }: DefineRangeFormProps): Promise<void> => {
    const startTS = new TimeStamp(start, "UTC");
    const endTS = new TimeStamp(end, "UTC");
    name = name.trim();
    if (name.length === 0) name = range?.name!;
    // remove leading and trailing whitespace
    const key = isCreate ? uuidv4() : layoutKey;

    const persisted = savePermanently.current || isRemoteEdit;

    if (persisted && client != null) {
      try {
        setLoading(true);
        const rng = await client.ranges.create({
          name,
          timeRange: new TimeRange(startTS, endTS),
          key,
        });
        await rng.addLabel(...labels);
      } finally {
        setLoading(false);
      }
    }
    dispatch(
      add({
        ranges: [
          {
            variant: "static",
            name,
            timeRange: { start: startTS.valueOf(), end: endTS.valueOf() },
            key,
            persisted,
          },
        ],
      }),
    );
    onClose();
  };

  const formRef = useRef<HTMLFormElement>(null);

  return (
    <Align.Space className={CSS.B("range-edit-layout")} grow>
      <Align.Space
        el="form"
        ref={formRef}
        size="small"
        grow
        className="console-form"
        onSubmit={(e) => {
          e.preventDefault();
          void methods.handleSubmit(onSubmit)(e);
        }}
        justify="center"
        id="define-range"
        noValidate
      >
        <FormProvider {...methods}>
          <Input.HFItem<string> name="name">
            {(p) => (
              <Input.Text
                autoFocus
                level="h2"
                variant="natural"
                placeholder="Range Name"
                {...p}
              />
            )}
          </Input.HFItem>
          <Align.Space direction="x" size="large">
            <Input.HFItem<number> name="start" label="From">
              {(p) => <Input.DateTime level="h4" variant="natural" {...p} />}
            </Input.HFItem>
            <Text.WithIcon level="h4" startIcon={<Icon.Arrow.Right />} />
            <Input.HFItem<number> name="end" label="To">
              {(p) => <Input.DateTime level="h4" variant="natural" {...p} />}
            </Input.HFItem>
          </Align.Space>
        </FormProvider>
      </Align.Space>
      <Nav.Bar location="bottom" size={48}>
        <Nav.Bar.End style={{ padding: "1rem" }}>
          {isCreate && (
            <Button.Button
              onClick={() => {
                savePermanently.current = true;
                formRef.current?.requestSubmit();
              }}
              variant="outlined"
              disabled={client == null || loading}
              loading={loading}
            >
              Save to Synnax
            </Button.Button>
          )}
          <Button.Button onClick={() => formRef.current?.requestSubmit()}>
            Save Locally
          </Button.Button>
        </Nav.Bar.End>
      </Nav.Bar>
    </Align.Space>
  );
};
