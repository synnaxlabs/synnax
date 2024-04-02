import { DataType, Synnax, TimeSpan, TimeStamp } from "@synnaxlabs/client";

const client = new Synnax({
    host: "localhost",
    port: 9090,
    username: "synnax",
    password: "seldon",
    secure: false
});

const time_channel = await client.channels.create({
    name: "basic_read_write_time",
    isIndex: true,
    dataType: DataType.TIMESTAMP
});

const data_channel = await client.channels.create({
    name: "basic_read_write_data",
    isIndex: false,
    dataType: DataType.FLOAT32,
    index: time_channel.key,
});

const N_SAMPLES = 500;

const start = TimeStamp.now();

console.log(BigInt(start.valueOf().toString()) - BigInt(start.bigInt.toString()))

const data = Float32Array.from({ length: N_SAMPLES }, (_, i) => Math.sin(i / 100));
const time = BigInt64Array.from({ length: N_SAMPLES }, (_, i) => BigInt(start.add(TimeSpan.milliseconds(i)).valueOf()));
console.log(time[0], start, Number(time[0]) - start.valueOf())

console.log(time_channel.key, data_channel.key)

// await client.telem.write(time_channel.key, start, time);
// await client.telem.write(data_channel.key, start, data);

const w1 = await client.telem.newWriter({
    start,
    channels: time_channel.key,
});
try {
    await w1.write(time_channel.key, time);
    console.log(await w1.commit());
} finally {
    await w1.close();
}

const w = await client.telem.newWriter({
    start,
    channels: data_channel.key,
});
try {
    await w.write(data_channel.key, data);
    console.log(await w.commit());
} finally {
    await w.close();
}


// await client.telem.write(data_channel.key, start, data);

// console.log("HERE")

client.close();