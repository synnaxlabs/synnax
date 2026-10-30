import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Text } from "@synnaxlabs/lyra/text";

import { noop, Option, Row, Section } from "@/options/common";
import { Field } from "@/options/samples";

type Look = "today" | "boxed" | "grid";

const ROWS = [
  ["start_time", "2026-10-27T13:32:20Z"],
  ["limit", "100"],
  ["", ""],
  ["channel", ""],
];

const PLACEHOLDERS = {
  today: { key: "Key", value: "Value" },
  alt: { key: "limit…", value: "100…" },
};

const cellClass = (value: string): string =>
  value === "" ? "opt-cell-empty" : "opt-cell-filled";

const Grid = ({ look, rows }: { look: Look; rows: string[][] }) => {
  const ph = look === "today" ? PLACEHOLDERS.today : PLACEHOLDERS.alt;
  const variant = look === "boxed" ? "outlined" : undefined;
  return (
    <Input.Table value={rows} onChange={noop}>
      <Input.TableColumn name="Parameter" type="string">
        {(p) => (
          <Input.Text
            {...p}
            variant={variant ?? p.variant}
            placeholder={ph.key}
            className={cellClass(p.value)}
          />
        )}
      </Input.TableColumn>
      <Input.TableColumn name="Value" type="string">
        {(p) => (
          <Input.Text
            {...p}
            variant={variant ?? p.variant}
            placeholder={ph.value}
            className={cellClass(p.value)}
          />
        )}
      </Input.TableColumn>
    </Input.Table>
  );
};

const AddButton = ({ label }: { label: string }) => (
  <Button.Button size="small" variant="text" textColor={9} className="opt-add-btn">
    <Icon.Add />
    {label}
  </Button.Button>
);

const Empty = () => (
  <Flex.Box y gap="tiny" align="start" className="opt-table-empty">
    <Text.Text color={9}>No headers. Add one to send it with every request.</Text.Text>
    <AddButton label="Add header" />
  </Flex.Box>
);

const TableSample = ({ look }: { look: Look }) => (
  <Flex.Box x gap="huge" align="start">
    <Row label="Rows: two filled, one empty, one with an empty value">
      <div style={{ width: 460 }}>
        <Field label="Query parameters">
          {look === "today" ? (
            <Grid look={look} rows={ROWS} />
          ) : (
            <div className="opt-table-custom">
              <Grid look={look} rows={ROWS} />
              <AddButton label="Add parameter" />
            </div>
          )}
        </Field>
      </div>
    </Row>
    <Row label="Empty">
      <div style={{ width: 340 }}>
        <Field label="Headers">
          {look === "today" ? <Grid look={look} rows={[]} /> : <Empty />}
        </Field>
      </div>
    </Row>
  </Flex.Box>
);

export const TableInputSection = () => (
  <Section
    id="table"
    n={8}
    title="Table input look"
    summary="Sets how a key-value table shows its cells at rest, so filled, empty, and placeholder cells differ. Alternatives also keep the remove button visible on every row and give the empty table a sentence."
  >
    <Option
      letter="A"
      title="Today"
      desc="Shadow inputs with no edge at rest; placeholders close to real values. Remove shows only on hover. An empty table shows a heading and Add row."
    >
      <TableSample look="today" />
    </Option>
    <Option
      letter="B"
      title="Boxed cells"
      className="tb-b"
      desc="Every cell is an outlined 28 px input. Placeholders sit two steps lighter and end in …. Remove always visible; Add parameter under the rows."
    >
      <TableSample look="boxed" />
    </Option>
    <Option
      letter="C"
      title="Ruled grid, light cells"
      className="tb-c"
      desc="One ruled grid with a heading band. Filled cells sit on a light fill; empty cells stay clear with a lighter placeholder. Remove always visible."
    >
      <TableSample look="grid" />
    </Option>
  </Section>
);
