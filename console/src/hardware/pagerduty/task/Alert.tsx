// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type rack, type status, type Synnax as Client } from "@synnaxlabs/client";
import {
  Button,
  Component,
  Divider,
  Flex,
  Form as PForm,
  Header,
  Icon,
  List,
  Menu as PMenu,
  Rack,
  Select,
  Status,
  Text,
} from "@synnaxlabs/pluto";
import { id } from "@synnaxlabs/x";
import { type FC, useCallback, useState } from "react";

import { Menu } from "@/components";
import { Common } from "@/hardware/common";
import {
  ALERT_SCHEMAS,
  ALERT_TYPE,
  type AlertConfig,
  type AlertPayload,
  type AlertSchemas,
  ZERO_ALERT_CONFIG,
  ZERO_ALERT_PAYLOAD,
} from "@/hardware/pagerduty/task/types";
import { Selector } from "@/selector";

export const ALERT_LAYOUT: Common.Task.Layout = {
  ...Common.Task.LAYOUT,
  type: ALERT_TYPE,
  name: ZERO_ALERT_PAYLOAD.name,
  icon: "Logo.PagerDuty",
};

export const AlertSelectable = Selector.createSimpleItem({
  title: ZERO_ALERT_PAYLOAD.name,
  icon: <Icon.Logo.PagerDuty />,
  layout: ALERT_LAYOUT,
});

const Properties = () => (
  <Flex.Box x grow>
    <PForm.TextField
      inputProps={ROUTING_KEY_INPUT_PROPS}
      path="config.routingKey"
      label="Routing key"
      grow
      style={ROUTING_KEY_STYLE}
    />
    <PForm.Field<number> path="rackKey" label="Connect from" grow>
      {selectRackRenderProp}
    </PForm.Field>
    <Common.Task.Fields.AutoStart />
  </Flex.Box>
);

const ROUTING_KEY_STYLE = { maxWidth: "70rem" };
const ROUTING_KEY_INPUT_PROPS = { type: "password" };

const selectRackRenderProp = Component.renderProp(
  (p: Omit<Rack.SelectSingleProps, "variant">) => (
    <Rack.SelectSingle {...p} initialQuery={INITIAL_RACK_QUERY} />
  ),
);

const INITIAL_RACK_QUERY: rack.RetrieveArgs = { integration: "pagerduty" };

interface AlertDetailsProps {
  itemKey: string;
}

const AlertDetails = ({ itemKey }: AlertDetailsProps) => {
  const path = `config.alerts.${itemKey}`;
  return (
    <Flex.Box grow style={DETAILS_STYLE} gap="small">
      <Flex.Box x>
        <PForm.Field<string> path={`${path}.status`} label="Status" grow>
          {selectStatusRenderProp}
        </PForm.Field>
        <PForm.SwitchField
          path={`${path}.treatErrorAsCritical`}
          label="Treat error as critical"
        />
      </Flex.Box>
      <PForm.TextField path={`${path}.component`} label="Component" optional />
      <PForm.TextField path={`${path}.group`} label="Group" optional />
      <PForm.TextField path={`${path}.class`} label="Class" optional />
    </Flex.Box>
  );
};

const selectStatusRenderProp = Component.renderProp(
  (p: Omit<Status.SelectProps, "variant">) => <Status.Select {...p} />,
);

const DETAILS_STYLE = { padding: "2rem", overflowY: "auto" } as const;
const LIST_STYLE = { width: "300px", flexShrink: 0, overflowY: "auto" } as const;

const AlertListItem = (props: List.ItemProps<string>) => {
  const { itemKey } = props;
  const statusKey = PForm.useFieldValue<status.Key>(`config.alerts.${itemKey}.status`);
  const status = Status.useRetrieve(
    { key: statusKey },
    { addStatusOnFailure: false },
  ).data;
  const isNotDefined = status == null;
  return (
    <Select.ListItem {...props} justify="between" align="center" x>
      <Flex.Box x align="center" gap={4}>
        <Status.Indicator variant={isNotDefined ? "disabled" : status.variant} />
        <Text.Text
          level="p"
          weight={500}
          status={isNotDefined ? "disabled" : undefined}
        >
          {isNotDefined ? "New alert" : status.name}
        </Text.Text>
      </Flex.Box>
      <Common.Task.EnableDisableButton path={`config.alerts.${itemKey}.enabled`} />
    </Select.ListItem>
  );
};

const alertListItem = Component.renderProp(AlertListItem);

interface AlertContextMenuProps extends PMenu.ContextMenuMenuProps {
  onRemove: (keys: string[]) => void;
  onSetEnabled: (keys: string[], enabled: boolean) => void;
}

const AlertContextMenu = ({ keys, onRemove, onSetEnabled }: AlertContextMenuProps) => {
  const canRemove = keys.length > 0;
  const alerts = PForm.useFieldValue<AlertConfig[]>("config.alerts").filter((a) =>
    keys.includes(a.key),
  );
  const canDisable = alerts.some(({ enabled }) => enabled);
  const canEnable = alerts.some(({ enabled }) => !enabled);
  return (
    <PMenu.Menu level="small" gap="small">
      {canRemove && (
        <PMenu.Item itemKey="remove" onClick={() => onRemove(keys)}>
          <Icon.Close />
          Remove
        </PMenu.Item>
      )}
      {canRemove && <PMenu.Divider />}
      {canEnable && (
        <PMenu.Item itemKey="enable" onClick={() => onSetEnabled(keys, true)}>
          <Icon.Enable />
          Enable
        </PMenu.Item>
      )}
      {canDisable && (
        <PMenu.Item itemKey="disable" onClick={() => onSetEnabled(keys, false)}>
          <Icon.Disable />
          Disable
        </PMenu.Item>
      )}
      {(canDisable || canEnable) && <PMenu.Divider />}
      <Menu.ReloadConsoleItem />
    </PMenu.Menu>
  );
};

const Form: FC<Common.Task.FormProps<AlertSchemas>> = () => {
  const { data, push, remove } = PForm.useFieldList<string, AlertConfig>(
    "config.alerts",
  );
  const { set } = PForm.useContext();
  const [selected, setSelected] = useState<string[]>([]);
  const menuProps = PMenu.useContextMenu();

  const handleAdd = useCallback(() => {
    const alert: AlertConfig = { ...ZERO_ALERT_CONFIG, key: id.create() };
    push(alert);
    setSelected([alert.key]);
  }, [push]);

  const handleRemove = useCallback(
    (keys: string[]) => {
      remove(keys);
      setSelected([]);
    },
    [remove],
  );

  const handleSetEnabled = useCallback(
    (keys: string[], enabled: boolean) => {
      for (const key of keys) set(`config.alerts.${key}.enabled`, enabled);
    },
    [set],
  );

  return (
    <Flex.Box x grow empty>
      <Flex.Box direction="y" style={LIST_STYLE} empty>
        <Header.Header>
          <Header.Title weight={500} color={10}>
            Alerts
          </Header.Title>
          <Header.Actions>
            <Button.Button
              onClick={handleAdd}
              variant="text"
              contrast={2}
              tooltip="Add alert"
              sharp
            >
              <Icon.Add />
            </Button.Button>
          </Header.Actions>
        </Header.Header>
        <PMenu.ContextMenu
          {...menuProps}
          menu={(p) => (
            <AlertContextMenu
              {...p}
              onRemove={handleRemove}
              onSetEnabled={handleSetEnabled}
            />
          )}
        >
          <Select.Frame<string, AlertConfig>
            multiple
            data={data}
            value={selected}
            onChange={setSelected}
            replaceOnSingle
            allowNone={false}
            autoSelectOnNone
          >
            <List.Items<string, AlertConfig> full="y" onContextMenu={menuProps.open}>
              {alertListItem}
            </List.Items>
          </Select.Frame>
        </PMenu.ContextMenu>
      </Flex.Box>
      <Divider.Divider direction="y" />
      {selected.length > 0 && <AlertDetails itemKey={selected[0]} />}
    </Flex.Box>
  );
};

const getInitialValues: Common.Task.GetInitialValues<AlertSchemas> = ({ config }) => {
  const pld: AlertPayload = { ...ZERO_ALERT_PAYLOAD };
  if (config != null) {
    const parsed = ALERT_SCHEMAS.config.safeParse(config);
    if (parsed.success) pld.config = parsed.data;
  }
  return pld;
};

const onConfigure: Common.Task.OnConfigure<AlertSchemas["config"]> = async (
  _client: Client,
  config,
) => [config, 0];

export const Alert = Common.Task.wrapForm({
  Properties,
  Form,
  schemas: ALERT_SCHEMAS,
  type: ALERT_TYPE,
  getInitialValues,
  onConfigure,
});
