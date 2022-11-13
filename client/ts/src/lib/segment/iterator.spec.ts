import test from 'ava';

import { newClient } from '../../setupspecs';
import { ContiguityError } from '../errors';
import { DataType, Rate, TimeRange, TimeSpan } from '../telem';
import { randomTypedArray } from '../util/telem';

const client = newClient();

const newChannel = async () => {
	return await client.channel.create({
		name: 'test',
		nodeId: 1,
		rate: Rate.Hz(25),
		dataType: DataType.Float64,
	});
};

test('TypedIterator - basic iteration', async (t) => {
	const ch = await newChannel();
	const writer = await client.data.newWriter([ch.key]);
	const data = randomTypedArray(25, ch.dataType);
	try {
		await writer.write(ch.key, TimeSpan.Second, data);
		await writer.write(ch.key, TimeSpan.Seconds(2), data);
		await writer.write(ch.key, TimeSpan.Seconds(3), data);
	} finally {
		await writer.close();
	}
	const iterator = await client.data.newIterator(
		new TimeRange(TimeSpan.Zero, TimeSpan.Seconds(4)),
		[ch.key],
		false
	);
	try {
		t.true(await iterator.seekFirst());
		let c = 0;
		while (await iterator.next(TimeSpan.Seconds(1))) {
			c++;
			t.true((await iterator.value())[ch.key].view.length === 25);
		}
		t.true(c === 3);
	} finally {
		await iterator.close();
	}
});

test('Client - basic read', async (t) => {
	const ch = await newChannel();
	const writer = await client.data.newWriter([ch.key]);
	const data = randomTypedArray(25, ch.dataType);
	try {
		await writer.write(ch.key, TimeSpan.Second, data);
		await writer.write(ch.key, TimeSpan.Seconds(2), data);
		await writer.write(ch.key, TimeSpan.Seconds(3), data);
	} finally {
		await writer.close();
	}
	const resData = await client.data.read(ch.key, TimeSpan.Zero, TimeSpan.Seconds(4));
	resData?.slice(0, 25).forEach((v, i) => t.true(v === data[i]));
	t.true(resData?.length === 75);
});

test('Client - incontiguous read', async (t) => {
	const ch = await newChannel();
	const data = randomTypedArray(25, ch.dataType);
	await ch.write(TimeSpan.Zero, data);
	await ch.write(TimeSpan.Seconds(2), data);
	const err = await t.throwsAsync(async () => {
		await client.data.read(ch.key, TimeSpan.Zero, TimeSpan.Seconds(4));
	});
	t.true(err instanceof ContiguityError);
});
