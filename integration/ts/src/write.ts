import { framer, Synnax } from "@synnaxlabs/client"
import { TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import * as fs from 'fs';
import { exit } from "process";

class IndexWriterGroup {
	indexChannels: string[] = [];
	dataChannels: string[] = [];

	constructor(indexChannels: string[], dataChannels: string[]) {
		this.indexChannels = indexChannels;
		this.dataChannels = dataChannels;
	}

	together(): string[] {
		return [...this.indexChannels, ...this.dataChannels];
	}
}

class TestConfig {
	identifier: string = "";
	numWriters: number = 0;
	domains: number = 0;
	samplesPerDomain: number = 0;
	timeRange: TimeRange = TimeRange.ZERO;
	autoCommit: boolean = false;
	indexPersistInterval: TimeSpan = TimeSpan.seconds(1);
	writerMode: framer.WriterMode = framer.WriterMode.PersistStream;
	channels: IndexWriterGroup[] = [];

	numChannels: () => number = () => {
		return this.channels.reduce((a, l: IndexWriterGroup) => a + l.together().length, 0);
	};

	constructor(
		identifier: string,
		numWriters: number,
		domains: number,
		samplesPerDomain: number,
		timeRangeStart: bigint,
		timeRangeEnd: bigint,
		autoCommit: boolean,
		indexPersistInterval: TimeSpan,
		writerMode: framer.WriterMode,
		channels: IndexWriterGroup[],
	) {
		this.identifier = identifier;
		this.numWriters = numWriters;
		this.domains = domains;
		this.samplesPerDomain = samplesPerDomain;
		this.timeRange = new TimeRange(timeRangeStart, timeRangeEnd);
		this.autoCommit = autoCommit;
		this.indexPersistInterval = indexPersistInterval;
		this.writerMode = writerMode;
		this.channels = channels;
	};
}

const client = new Synnax({
	host: "localhost",
	port: 9090,
	username: "synnax",
	password: "seldon",
	secure: false,
});


class WriteTest {
	tc: TestConfig;
	constructor(argv: string[]) {
		let argvCounter = 2;
		const identifier = argv[argvCounter++];
		const numWriters = parseInt(argv[argvCounter++]);
		const domains = parseInt(argv[argvCounter++]);
		const samplesPerDomain = parseInt(argv[argvCounter++]);
		const timeRangeStart = BigInt(argv[argvCounter++]);
		const timeRangeEnd = BigInt(argv[argvCounter++]);
		const autoCommit = argv[argvCounter++] === "true";
		const indexPersistInterval = new TimeSpan(BigInt(argv[argvCounter++]));
		const writerMode = parseInt(argv[argvCounter++]) as framer.WriterMode;
		const numberOfChannelGroups = parseInt(argv[argvCounter++]);
		const channelGroups: IndexWriterGroup[] = [];

		for (let i = 0; i < numberOfChannelGroups; i++) {
			const numberOfIndex = parseInt(argv[argvCounter++]);
			const numberOfData = parseInt(argv[argvCounter++]);
			const indexChannels = argv.slice(argvCounter, argvCounter + numberOfIndex);
			argvCounter += numberOfIndex;
			const dataChannels = argv.slice(argvCounter, argvCounter + numberOfData);
			argvCounter += numberOfData;
			channelGroups.push(new IndexWriterGroup(indexChannels, dataChannels));
		}

		this.tc = new TestConfig(
			identifier,
			numWriters,
			domains,
			samplesPerDomain,
			timeRangeStart,
			timeRangeEnd,
			autoCommit,
			indexPersistInterval,
			writerMode,
			channelGroups,
		);
	}

	async testWithTiming(): Promise<void> {
		const start = TimeStamp.now();
		await this.test();
		const end = TimeStamp.now();

		const time: TimeSpan = start.span(end);
		const samples = this.tc.numChannels() * this.tc.samplesPerDomain * this.tc.domains;
		const samplesPerSecond = samples / (Number(time) / Number(TimeSpan.SECOND));
		const s = `
-- TypeScript Write (${this.tc.identifier}) --
Samples written: ${samples}
Time taken: ${time}
Calculated Samples per Second: ${samplesPerSecond.toFixed(2)}
Configuration:
    Number of writers: ${this.tc.numWriters}
    Number of channels: ${this.tc.numChannels()}
    Number of domains: ${this.tc.domains}
    Samples per domain: ${this.tc.samplesPerDomain}
    Auto commit: ${this.tc.autoCommit}
    Index persist interval: ${this.tc.indexPersistInterval}
    Writer mode: ${framer.WriterMode[this.tc.writerMode]}

`;

		fs.appendFileSync("timing.log", s);
	}

	async test(): Promise<void> {
		const writers = new Array(this.tc.numWriters).fill(null);
		const timeSpanPerDomain = Number(this.tc.timeRange.span) / this.tc.domains;

		for (let i = 0; i < this.tc.numWriters; i++) {
			writers[i] = await client.openWriter({
				start: this.tc.timeRange.start,
				channels: this.tc.channels[i].together(),
				mode: this.tc.writerMode,
				enableAutoCommit: this.tc.autoCommit,
				autoIndexPersistInterval: this.tc.indexPersistInterval,
			});
		}

		try {
			const tsHwm = this.tc.timeRange.start.add(new TimeSpan(1));
			for (let i = 0; i < this.tc.domains; i++) {
				const timestamps = Array.from<any, bigint>(
					{ length: this.tc.samplesPerDomain },
					(_, k) => tsHwm.valueOf() + BigInt(k * timeSpanPerDomain) / BigInt(this.tc.samplesPerDomain));
				const data = timestamps.map(ts => Math.sin(0.0000000001 * Number(ts)));

				for (let j = 0; j < writers.length; j++) {
					const writer = writers[j];
					const dataDict: { [key: string]: number[] | bigint[] } = {};

					this.tc.channels[j].indexChannels.forEach(indexChannel => {
						dataDict[indexChannel] = timestamps;
					});
					this.tc.channels[j].dataChannels.forEach(dataChannel => {
						dataDict[dataChannel] = data;
					});

					await writer.write(dataDict);

					if (!this.tc.autoCommit) {
						await writer.commit();
					}
				}

				tsHwm.add(new TimeSpan(timeSpanPerDomain + 1));
			}
		} finally {
			for (const writer of writers) {
				await writer.close();
			}
		}
	}
}

async function main(): Promise<void> {
	await new WriteTest(process.argv).testWithTiming().catch(error => {
		console.error(error)
		client.close()
		exit(1)
	})

	client.close()
}

await main()
