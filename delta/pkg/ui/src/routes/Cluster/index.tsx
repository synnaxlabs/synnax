import { Header, Space } from "@arya-analytics/pluto";
import { Atom2, Grain } from "tabler-icons-react";
import Sidebar from "../../lib/Sidebar/Sidebar";
import { HexagonBar } from "@arya-analytics/pluto";
import { Statistic } from "@arya-analytics/pluto/src";

const dummyMetrics = () => {
  return [
    {
      name: "Memory",
      value: Math.random() * 100,
      max: 100,
      units: "GB",
    },
    {
      name: "CPU",
      value: Math.random() * 100,
      max: 100,
      units: "%",
    },
    {
      name: "CPU",
      value: Math.random() * 100,
      max: 100,
      units: "%",
    },
    {
      name: "Net",
      value: Math.random() * 100,
      max: 100,
      units: "%",
    },
  ];
};

export default function Cluster() {
  document.title = "Delta - Cluster Overview";
  return (
    <Space empty direction="horizontal">
      <Sidebar />
      <Space empty direction="vertical" style={{ flexGrow: 1 }} align="stretch">
        <Header size="h3" text="Cluster Overview" icon={<Grain />} />
        <Space empty direction="horizontal" style={{ flexGrow: 1 }}>
          <Space
            empty
            direction="vertical"
            style={{ flexGrow: 1 }}
            size="large"
          >
            <Header size="h4" text="Nodes" icon={<Atom2 />} />
            <Space
              direction="horizontal"
              size="medium"
              style={{ flexWrap: "wrap", flexGrow: 1, overflow: "auto" }}
              justify="center"
              align="center"
            >
              {Array.from({ length: 3 }).map((_, i) => (
                <HexagonBar
                  key={i}
                  strokeWidth={5}
                  width={250}
                  metrics={dummyMetrics()}
                />
              ))}
            </Space>
            <Space
              direction="horizontal"
              size={30}
              justify="center"
              align="center"
              style={{
                borderTop: "var(--border-width) solid var(--gray-m2)",
                flexShrink: 0,
                padding: 20,
              }}
            >
              <Statistic value={6} label="Live Nodes" level="h1" />
              <Statistic
                value={3}
                label="Dead Nodes"
                level="h1"
                variant="error"
              />
              <Statistic
                value={5}
                label="Warning Nodes"
                level="h1"
                variant="primary"
              />
            </Space>
          </Space>
        </Space>
      </Space>
    </Space>
  );
}
