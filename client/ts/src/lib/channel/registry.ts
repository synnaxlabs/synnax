import { ChannelPayload } from './payload';
import Retriever from './retriever';

export default class Registry {
	private retriever: Retriever;
	private channels: Map<string, ChannelPayload>;

	constructor(retriever: Retriever) {
		this.retriever = retriever;
		this.channels = new Map();
	}

	async get(key: string): Promise<ChannelPayload> {
		let channel = this.channels.get(key);
		if (channel === undefined) {
			channel = await this.retriever.retrieve({ key });
			this.channels.set(key, channel);
		}
		return channel;
	}

	async getN(...keys: string[]): Promise<ChannelPayload[]> {
		const results: ChannelPayload[] = [];
		const retrieveKeys: string[] = [];
		keys.forEach((key) => {
			const channel = this.channels.get(key);
			if (channel === undefined) retrieveKeys.push(key);
			else results.push(channel);
		});
		if (retrieveKeys.length > 0) {
			const channels = await this.retriever.filter({ keys: retrieveKeys });
			channels.forEach((channel) => {
				this.channels.set(channel.key as string, channel);
				results.push(channel);
			});
		}
		return results;
	}
}
