import test from 'ava';

import { newClient } from '../../setupspecs';
import { ContiguityError, ValidationError } from '../errors';
import { DataType, Rate, TimeSpan } from '../telem';
import { randomTypedArray } from '../util/telem';

const client = newClient();

const newChannel = async () => {
	return await client.channel.create({
		name: 'test',
		nodeId: 1,
		rate: Rate.Hz(1),
		dataType: DataType.Float64,
	});
};

test('TypedWriter - basic write', async (t) => {
	const ch = await newChannel();
	const writer = await client.data.newWriter([ch.key]);
	try {
		await writer.write(ch.key, 0, randomTypedArray(10, ch.dataType));
	} finally {
		await writer.commit();
		await writer.close();
	}
	t.true(true);
});

test('TypedWriter - invalid data type', async (t) => {
	const ch = await newChannel();
	const writer = await client.data.newWriter([ch.key]);
	try {
		await writer.write(ch.key, 0, randomTypedArray(16, DataType.Uint8));
		t.fail('Expected error');
	} catch (err) {
		t.true(err instanceof ValidationError);
	} finally {
		await writer.commit();
		await writer.close();
	}
});

test('TypedWriter - non contiguous', async (t) => {
	const ch = await newChannel();
	const writer = await client.data.newWriter([ch.key]);
	try {
		await writer.write(ch.key, 0, randomTypedArray(10, ch.dataType));
		await writer.write(ch.key, 12, randomTypedArray(10, ch.dataType));
		t.fail('Expected error');
	} catch (err) {
		t.true(err instanceof ContiguityError);
	} finally {
		await writer.close();
	}
});

test('TypedWriter - multi segment write', async (t) => {
	const ch = await newChannel();
	const nSamples = 1000;
	const nWrites = 100;
	const writer = await client.data.newWriter([ch.key]);
	const data = randomTypedArray(nSamples, ch.dataType);
	try {
		for (let i = 0; i < nWrites; i++) {
			await writer.write(ch.key, TimeSpan.Seconds(i * nSamples), data);
		}
	} finally {
		await writer.close();
	}
	t.true(true);
});
