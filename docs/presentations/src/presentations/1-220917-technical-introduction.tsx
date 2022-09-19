import {
  Presentation,
  SlideContainer,
  Node,
  usePresentationContext, HorizontalPhaser
} from "../components";
import LogoSlide from "./common/LogoSlide";
import BulletSlide, { Bullet } from "./common/BulletSlide";
import { Space, useThemeContext, Text } from "@synnaxlabs/pluto";
import SplitSlide from "./common/SplitSlide";
import HeaderSlide from "./common/HeaderSlide";
import { GiBirchTrees, GiStrikingDiamonds } from "react-icons/gi";
import CenteredSlide from "./common/CenteredSlide";
import {motion} from "framer-motion";
import PhaserPath from "../components/PhaserPath";



const SoftwareLayerBullet = ({ index }: { index: number }) => {
  const { theme } = useThemeContext();
  return (
    <div
      style={{
        width: 120,
        height: 30,
        background: theme.colors.visualization.palettes.default[index],
      }}
    />
  );
};

export default function P() {
  const { theme } = useThemeContext();
  return (
    <Presentation>
      <SlideContainer
          title={"High Level Software for Hardware"}
      >
        <CenteredSlide>
          <Text level="h1">High Level Software for Hardware</Text>
        </CenteredSlide>
      </SlideContainer>
      <SlideContainer
        title="The Problem"
        >
        <CenteredSlide>
          <Text level="h1">Closing the hardware iteration feedback loop.</Text>
        </CenteredSlide>
        </SlideContainer>
      <SlideContainer title="Technical Introduction" logoColor={false}>
        <LogoSlide />
      </SlideContainer>
      <SlideContainer title="Objectives" transitionCount={4} logoColor="white">
        <BulletSlide theme="primary">
          <Bullet>What is Synnax?</Bullet>
          <Bullet>How does it work?</Bullet>
          <Bullet>Why should I work on it?</Bullet>
          <Bullet>How do I get started?</Bullet>
        </BulletSlide>
      </SlideContainer>
      <SlideContainer
        title={"What is Synnax?"}
        transitionCount={5}
      >
        <SplitSlide widths={[700]}>
          <Space direction="vertical" justify="center" align="center" empty>
            <Space direction="horizontal" size={16}>
              <Node number={1} />
              <Node number={2} />
            </Space>
            <Node number={3} />
          </Space>
          <BulletSlide title="Distributed Telemetry Engine">
            <Bullet>
              Connect to and communicate with DAQ hardware/software.
            </Bullet>
            <Bullet>
              Write to and read from a time-series optimized data store.
            </Bullet>
            <Bullet>Relay telemetry between nodes and to clients.</Bullet>
            <Bullet>Run analysis on both real time and historical data.</Bullet>
          </BulletSlide>
        </SplitSlide>
      </SlideContainer>
      <SlideContainer title="Node Software Architecture" transitionCount={5}>
        <SplitSlide widths={[900]}>
          <BulletSlide title="Node Software Architecture">
            <Bullet
              bullet={<SoftwareLayerBullet index={0} />}
              subText={"Reads and writes telemetry to disk."}
            >
              Storage Layer
            </Bullet>
            <Bullet
              bullet={<SoftwareLayerBullet index={1} />}
              subText={
                "Builds overlay network and exposes cluster as a monolithic data space."
              }
            >
              Distribution Layer
            </Bullet>
            <Bullet
              bullet={<SoftwareLayerBullet index={2} />}
              subText={
                "Performs services such as computation, aggregation, authentication, etc."
              }
            >
              Service Layer
            </Bullet>
            <Bullet
              bullet={<SoftwareLayerBullet index={3} />}
              subText={
                "Exposes an interface for clients to interact with the node."
              }
            >
              API Layer
            </Bullet>
          </BulletSlide>
          <Space direction="horizontal" justify="center">
            <Node width={700} />
          </Space>
        </SplitSlide>
      </SlideContainer>
      <SlideContainer
        title={"Storage Layer - Workloads"}
        transitionCount={3}
      >
        <StorageLayerWorkloads />
      </SlideContainer>
      <SlideContainer
        title={"Storage Layer - Engines"}
        transitionCount={4}
      >
        <HeaderSlide
          title={"Storage Layer - Engines"}
          color={theme.colors.visualization.palettes.default[0]}
          textColor={"var(--white)"}
        >
          <SplitSlide justify="center" align="center">
            <BulletSlide
              title={
                <Space direction="vertical" align="center" justify="center">
                  <GiStrikingDiamonds size={150} />
                  <Text level="h1">Cesium</Text>
                </Space>
              }
            >
              <Bullet>Key Value Store + Time Ordered File System</Bullet>
              <Bullet>Optimized for Long Lived, High Volume Queries</Bullet>
              <Bullet>Streaming Design</Bullet>
            </BulletSlide>
            <BulletSlide
              title={
                <Space justify="center" align="center">
                  <GiBirchTrees size={150} />
                  <h1>Aspen</h1>
                </Space>
              }
            >
              <Bullet>Key Value Store</Bullet>
              <Bullet>Optimized for Short, Single Value Queries</Bullet>
              <Bullet>Unary Design</Bullet>
            </BulletSlide>
          </SplitSlide>
        </HeaderSlide>
      </SlideContainer>
      <SlideContainer
        title={"Distribution Layer"}
        transitionCount={2}
      >
        <DistributionLayerIntroduction />
      </SlideContainer>
      <SlideContainer
        title={"Distribution Layer | Cluster Topology"}
        transitionCount={4}
      >
        <HeaderSlide
          title={"Distribution Layer - Cluster Topology"}
          color={theme.colors.visualization.palettes.default[1]}
          textColor={"var(--white)"}
        >
          <SplitSlide justify="center" align="center">
            <Space size={16} justify="center" align="center">
              <Space direction="horizontal" size={16}>
                <Node number={1} />
                <Node number={2} />
              </Space>
              <Node number={3} />
              <motion.svg style={{position: "absolute" }} height={400} width={450}>
                <HorizontalPhaser color={"var(--primary-z)"} top={20} width={140} left={155} />
                <PhaserPath
                  start={[40, 150]}
                  points={[
                    [40, 360, false],
                    [125, 360, false],
                  ]}
                  strokeWidth={16}
                  spacing={5}
                  count={1}
                  speed={1}
                  endLine={false}
                  startLine={false}
                  color={"var(--primary-z)"}
                />
                <PhaserPath
                  start={[430, 150]}
                  points={[
                    [430, 360, false],
                    [330, 360, false],
                  ]}
                  strokeWidth={16}
                  spacing={5}
                  count={1}
                  speed={1}
                  endLine={false}
                  startLine={false}
                  color={"var(--gray-m2)"}
                />
              </motion.svg>
            </Space>
            <Space
              direction="vertical"
              size={16}
            >
              <BulletSlide title="Nodes are Constantly Propagating Information">
                <Bullet>Cluster Membership</Bullet>
                <Bullet>Cluster State</Bullet>
                <Bullet>Cluster Metadata</Bullet>
              </BulletSlide>
            </Space>
          </SplitSlide>
        </HeaderSlide>
      </SlideContainer>
      <SlideContainer title={"Distribution Layer - Reads and Writes"} >
        <HeaderSlide
          title={"Distribution Layer - Reads and Writes"}
          color={theme.colors.visualization.palettes.default[1]}
          textColor={"var(--white)"}
          >
          <CenteredSlide>
            <Text level="h1">Leveraging Cluster Topology to Balance Telemetry Storage</Text>
          </CenteredSlide>
        </HeaderSlide>
      </SlideContainer>
      <SlideContainer
        title={"Service Layer"}
        transitionCount={6}
      >
        <HeaderSlide
          title={"Service Layer"}
          color={theme.colors.visualization.palettes.default[2]}
          textColor={"var(--white)"}
        >
          <BulletSlide
            title={"Core Data Processing"}
          >
            <Bullet>Authentication</Bullet>
            <Bullet>Authorization</Bullet>
            <Bullet>Validation</Bullet>
            <Bullet>Computation + Aggregation</Bullet>
            <Bullet>Integrity Checks</Bullet>
          </BulletSlide>
        </HeaderSlide>
      </SlideContainer>
      <SlideContainer
        title={"Interface Layer"}
      >
        <HeaderSlide
          title={"Interface Layer"}
          color={theme.colors.visualization.palettes.default[3]}
          textColor={"var(--white)"}
        >
          <CenteredSlide>
            <Text level="h1">Expose data services to the consumer in a meaningful way</Text>
          </CenteredSlide>
        </HeaderSlide>
      </SlideContainer>
      <SlideContainer
        title={"Why Should I Work on this Project?"}
        transitionCount={6}
        logoColor="white"
      >
        <BulletSlide
          title="Why Should You Work on this Project?"
          extra={<Text level="h3" style={{color: "var(--white)"}}>And high level software for hardware in general..</Text>}
          theme="primary"
        >
          <Bullet>Have direct, broad impact.</Bullet>
          <Bullet>Build software that makes people's lives better</Bullet>
          <Bullet>Learn how to write distributed, big data systems.</Bullet>
        </BulletSlide>
      </SlideContainer>
    </Presentation>
  );
}

const StorageLayerWorkloads = () => {
  const { theme } = useThemeContext();
  const { transition } = usePresentationContext();
  return (<HeaderSlide
          title={"Storage Layer - Workloads"}
          color={theme.colors.visualization.palettes.default[0]}
          textColor={"var(--white)"}
        >
          <SplitSlide justify="start" align="start">
            <Space
              align="center"
              size={45}
              style={{ marginTop: 150, height: 800 }}
              grow
            >
              {[<Text level="h1">Telemetry</Text>,
              <Text level="h2">[1000.234, 2.312, 3.123, 1500.17, ...]</Text>,
              <Space align="center" justify="center" size="large">
                <Text level="h2">High Volume</Text>
                <Text level="h2">Append Only</Text>
                <Text level="h2">Simple, Predictable Retrievals</Text>
              </Space>].map((child, index) => (index <= transition && child))}
            </Space>
            <Space
              align="center"
              size={28}
              style={{ marginTop: 150, height: 800 }}
              grow
            >
              {[<Text level="h1">Meta Data + State</Text>,
              <Text level="h2">
                {"{"} <br />
                <span style={{ marginLeft: 40 }} />
                name: "myTemperatureSensor", <br />
                <span style={{ marginLeft: 40 }} />
                units: "C", <br />
                <span style={{ marginLeft: 40 }} />
                dataRate: 25Hz, <br />
                <span style={{ marginLeft: 40 }} />
                ...
                <br />
                {"}"}
              </Text>,
              <Space align="center" justify="center" size="large">
                <Text level="h2">Low Volume</Text>
                <Text level="h2">Frequent Updates</Text>
                <Text level="h2">Complex, Dynamic Retrievals</Text>
              </Space>].map((child, index) => (index <= transition && child))}
            </Space>
          </SplitSlide>
        </HeaderSlide>
  );
}

const DistributionLayerIntroduction = ()  => {
  const {transition} = usePresentationContext()
  const {theme} = useThemeContext()
  return (<HeaderSlide
          title={"Distribution Layer"}
          color={theme.colors.visualization.palettes.default[1]}
          textColor={"var(--white)"}
        >
            <CenteredSlide>
              <Text level="h1">
                Expose the cluster as a monolithic data space.
              </Text>
              {transition > 0 && (
                <>
              <Text level="h1">=</Text>
              <Text level="h1">
                A client can execute the same query against any two nodes and receive the same result.
              </Text>
                </>
              )}
            </CenteredSlide>
        </HeaderSlide>
  )
}
