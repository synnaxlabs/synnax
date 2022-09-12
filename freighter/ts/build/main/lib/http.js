"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.POSTClient = exports.GETClient = void 0;
const axios_1 = __importDefault(require("axios"));
const errors_1 = require("./errors");
class HTTPClient {
    constructor(endpoint, encoder) {
        this.endpoint = endpoint;
        this.encoder = encoder;
    }
    get() {
        return new GETClient(this.endpoint, this.encoder);
    }
    post() {
        return new POSTClient(this.endpoint, this.encoder);
    }
}
exports.default = HTTPClient;
class Core {
    constructor(endpoint, encoder) {
        this.endpoint = endpoint.child({ protocol: 'http' });
        this.encoder = encoder;
    }
    get headers() {
        return {
            'Content-Type': this.encoder.contentType,
        };
    }
    requestConfig() {
        return {
            headers: this.headers,
            responseType: 'arraybuffer',
        };
    }
}
class GETClient extends Core {
    async send(target, req) {
        const queryString = buildQueryString(req);
        const url = this.endpoint.path(target) + '?' + queryString;
        const response = await axios_1.default.get(url, this.requestConfig());
        if (response.status !== 200) {
            const err = this.encoder.decode(response.data);
            return [undefined, (0, errors_1.decodeError)(err)];
        }
        const data = this.encoder.decode(response.data);
        return [data, undefined];
    }
}
exports.GETClient = GETClient;
class POSTClient extends Core {
    async send(target, req) {
        const url = this.endpoint.path(target);
        const response = await axios_1.default.post(url, this.encoder.encode(req), this.requestConfig());
        if (response.status !== 200) {
            const err = this.encoder.decode(response.data);
            return [undefined, (0, errors_1.decodeError)(err)];
        }
        const data = this.encoder.decode(response.data);
        return [data, undefined];
    }
}
exports.POSTClient = POSTClient;
const buildQueryString = (request) => {
    const query = Object.keys(request)
        .map((key) => `${key}=${request[key]}`)
        .join('&');
    return query;
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiaHR0cC5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbIi4uLy4uLy4uL3NyYy9saWIvaHR0cC50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiOzs7Ozs7QUFBQSxrREFBa0Q7QUFJbEQscUNBQXFEO0FBR3JELE1BQXFCLFVBQVU7SUFJN0IsWUFBWSxRQUFrQixFQUFFLE9BQXVCO1FBQ3JELElBQUksQ0FBQyxRQUFRLEdBQUcsUUFBUSxDQUFDO1FBQ3pCLElBQUksQ0FBQyxPQUFPLEdBQUcsT0FBTyxDQUFDO0lBQ3pCLENBQUM7SUFFRCxHQUFHO1FBQ0QsT0FBTyxJQUFJLFNBQVMsQ0FBQyxJQUFJLENBQUMsUUFBUSxFQUFFLElBQUksQ0FBQyxPQUFPLENBQUMsQ0FBQztJQUNwRCxDQUFDO0lBRUQsSUFBSTtRQUNGLE9BQU8sSUFBSSxVQUFVLENBQUMsSUFBSSxDQUFDLFFBQVEsRUFBRSxJQUFJLENBQUMsT0FBTyxDQUFDLENBQUM7SUFDckQsQ0FBQztDQUNGO0FBaEJELDZCQWdCQztBQUVELE1BQU0sSUFBSTtJQUlSLFlBQVksUUFBa0IsRUFBRSxPQUF1QjtRQUNyRCxJQUFJLENBQUMsUUFBUSxHQUFHLFFBQVEsQ0FBQyxLQUFLLENBQUMsRUFBRSxRQUFRLEVBQUUsTUFBTSxFQUFFLENBQUMsQ0FBQztRQUNyRCxJQUFJLENBQUMsT0FBTyxHQUFHLE9BQU8sQ0FBQztJQUN6QixDQUFDO0lBRUQsSUFBSSxPQUFPO1FBQ1QsT0FBTztZQUNMLGNBQWMsRUFBRSxJQUFJLENBQUMsT0FBTyxDQUFDLFdBQVc7U0FDekMsQ0FBQztJQUNKLENBQUM7SUFFRCxhQUFhO1FBQ1gsT0FBTztZQUNMLE9BQU8sRUFBRSxJQUFJLENBQUMsT0FBTztZQUNyQixZQUFZLEVBQUUsYUFBYTtTQUM1QixDQUFDO0lBQ0osQ0FBQztDQUNGO0FBRUQsTUFBYSxTQUFVLFNBQVEsSUFBSTtJQUNqQyxLQUFLLENBQUMsSUFBSSxDQUNSLE1BQWMsRUFDZCxHQUFPO1FBRVAsTUFBTSxXQUFXLEdBQUcsZ0JBQWdCLENBQUMsR0FBOEIsQ0FBQyxDQUFDO1FBQ3JFLE1BQU0sR0FBRyxHQUFHLElBQUksQ0FBQyxRQUFRLENBQUMsSUFBSSxDQUFDLE1BQU0sQ0FBQyxHQUFHLEdBQUcsR0FBRyxXQUFXLENBQUM7UUFDM0QsTUFBTSxRQUFRLEdBQUcsTUFBTSxlQUFLLENBQUMsR0FBRyxDQUFDLEdBQUcsRUFBRSxJQUFJLENBQUMsYUFBYSxFQUFFLENBQUMsQ0FBQztRQUU1RCxJQUFJLFFBQVEsQ0FBQyxNQUFNLEtBQUssR0FBRyxFQUFFO1lBQzNCLE1BQU0sR0FBRyxHQUFHLElBQUksQ0FBQyxPQUFPLENBQUMsTUFBTSxDQUFlLFFBQVEsQ0FBQyxJQUFJLENBQUMsQ0FBQztZQUM3RCxPQUFPLENBQUMsU0FBUyxFQUFFLElBQUEsb0JBQVcsRUFBQyxHQUFHLENBQUMsQ0FBQyxDQUFDO1NBQ3RDO1FBRUQsTUFBTSxJQUFJLEdBQUcsSUFBSSxDQUFDLE9BQU8sQ0FBQyxNQUFNLENBQUssUUFBUSxDQUFDLElBQUksQ0FBQyxDQUFDO1FBQ3BELE9BQU8sQ0FBQyxJQUFJLEVBQUUsU0FBUyxDQUFDLENBQUM7SUFDM0IsQ0FBQztDQUNGO0FBakJELDhCQWlCQztBQUVELE1BQWEsVUFBVyxTQUFRLElBQUk7SUFDbEMsS0FBSyxDQUFDLElBQUksQ0FDUixNQUFjLEVBQ2QsR0FBTztRQUVQLE1BQU0sR0FBRyxHQUFHLElBQUksQ0FBQyxRQUFRLENBQUMsSUFBSSxDQUFDLE1BQU0sQ0FBQyxDQUFDO1FBQ3ZDLE1BQU0sUUFBUSxHQUFHLE1BQU0sZUFBSyxDQUFDLElBQUksQ0FDL0IsR0FBRyxFQUNILElBQUksQ0FBQyxPQUFPLENBQUMsTUFBTSxDQUFDLEdBQUcsQ0FBQyxFQUN4QixJQUFJLENBQUMsYUFBYSxFQUFFLENBQ3JCLENBQUM7UUFFRixJQUFJLFFBQVEsQ0FBQyxNQUFNLEtBQUssR0FBRyxFQUFFO1lBQzNCLE1BQU0sR0FBRyxHQUFHLElBQUksQ0FBQyxPQUFPLENBQUMsTUFBTSxDQUFlLFFBQVEsQ0FBQyxJQUFJLENBQUMsQ0FBQztZQUM3RCxPQUFPLENBQUMsU0FBUyxFQUFFLElBQUEsb0JBQVcsRUFBQyxHQUFHLENBQUMsQ0FBQyxDQUFDO1NBQ3RDO1FBRUQsTUFBTSxJQUFJLEdBQUcsSUFBSSxDQUFDLE9BQU8sQ0FBQyxNQUFNLENBQUssUUFBUSxDQUFDLElBQUksQ0FBQyxDQUFDO1FBQ3BELE9BQU8sQ0FBQyxJQUFJLEVBQUUsU0FBUyxDQUFDLENBQUM7SUFDM0IsQ0FBQztDQUNGO0FBcEJELGdDQW9CQztBQUVELE1BQU0sZ0JBQWdCLEdBQUcsQ0FBQyxPQUFnQyxFQUFFLEVBQUU7SUFDNUQsTUFBTSxLQUFLLEdBQUcsTUFBTSxDQUFDLElBQUksQ0FBQyxPQUFPLENBQUM7U0FDL0IsR0FBRyxDQUFDLENBQUMsR0FBRyxFQUFFLEVBQUUsQ0FBQyxHQUFHLEdBQUcsSUFBSSxPQUFPLENBQUMsR0FBRyxDQUFDLEVBQUUsQ0FBQztTQUN0QyxJQUFJLENBQUMsR0FBRyxDQUFDLENBQUM7SUFDYixPQUFPLEtBQUssQ0FBQztBQUNmLENBQUMsQ0FBQyJ9