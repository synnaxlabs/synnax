import { TimeStamp, TimeSpan, framer } from "@synnaxlabs/client";
import * as fs from 'fs';

const FILE_NAME = "timing.log"

function TimeWrite() {
    return function (target: any, propertyKey: string, descriptor: PropertyDescriptor) {
        const originalMethod = descriptor.value;

        descriptor.value = async function (...args: any[]) {
            const start = TimeStamp.now();
            await originalMethod.apply(this, args);
            const end = TimeStamp.now();

            const time: TimeSpan = start.span(end);
            const params = args[0];
            const samples = params.numChannels() * params.samplesPerDomain * params.domains;
            const samplesPerSecond = samples / (Number(time) / Number(TimeSpan.SECOND));
            const s = `
-- TypeScript Write (${params.identifier}) --
Samples written: ${samples}
Time taken: ${time}
Calculated Samples per Second: ${samplesPerSecond.toFixed(2)}
Configuration:
    Number of writers: ${params.numWriters}
    Number of channels: ${params.numChannels()}
    Number of domains: ${params.domains}
    Samples per domain: ${params.samplesPerDomain}
    Auto commit: ${params.autoCommit}
    Index persist interval: ${params.indexPersistInterval}
    Writer mode: ${framer.WriterMode[params.writerMode]}

`;

            fs.appendFileSync(FILE_NAME, s);
        };
    };
}

function timeRead() {
    return function (target: any, propertyKey: string, descriptor: PropertyDescriptor) {
        const originalMethod = descriptor.value;

        descriptor.value = async function (...args: any[]) {
            const start = TimeStamp.now();
            const samples = await originalMethod.apply(this, args);
            const end = TimeStamp.now();

            const time: TimeSpan = start.span(end);
            const params = args[0];
            const samplesPerSecond = samples / (Number(time) / Number(TimeSpan.SECOND));
            const s = `
-- TypeScript Read (${params.identifier}) --
Samples read: ${samples}
Time taken: ${time}
Calculated Samples per Second: ${samplesPerSecond.toFixed(2)}
Configuration:
    Number of iterators: ${params.numIterators}
    Number of channels: ${params.numChannels()}
    Chunk size: ${params.chunkSize}

`;

            fs.appendFileSync(FILE_NAME, s);
        };

        return descriptor;
    };
}

function timeDelete() {
    return function (target: any, _context: ClassMethodDecoratorContext) {
        return async function (this: any, ...args: any[]) {
            const start = TimeStamp.now();
            await target.call(this, ...args);
            const end = TimeStamp.now();

            const time: TimeSpan = start.span(end);
            const params = args[0];
            const s = `
-- TypeScript Delete (${params.identifier}) --
Time taken: ${time}
Configuration:
    Number of channels: ${params.channels.length}
    Time Range: ${params.timeRange}

`;

            fs.appendFileSync(FILE_NAME, s);
        };
    };
}

function TimeStream(target: any, _context: ClassMethodDecoratorContext) {
    return async function (this: any, ...args: any[]) {
        const start = TimeStamp.now();
        const samples = await target.call(this, ...args);
        const end = TimeStamp.now();

        const time: TimeSpan = start.span(end);
        const params = args[0];
        const samplesPerSecond = samples / (Number(time) / Number(TimeSpan.SECOND));
        const s = `
-- TypeScript Stream (${params.identifier}) --
Samples streamed: ${samples}
Time taken: ${time}
Calculated Samples per Second: ${samplesPerSecond.toFixed(2)}
Configuration:
    Number of streamers: 1
    Number of channels: ${params.channels.length}
    Close after frames: ${params.closeAfterFrames}

`;

        fs.appendFileSync(FILE_NAME, s);
        return samples;
    };
};

export { TimeWrite, timeRead, timeDelete, TimeStream };
