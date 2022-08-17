import { EncoderDecoder as BaseEncoderDecoder } from "../encode/encoderDecoder";

enum Method {
  GET = "GET",
  POST = "POST",
  PUT = "PUT",
}

type Params = {
  [key: string]: string;
};

export class HTTPClient {
  public token: HeaderProvider | undefined;
  private ecd: EncoderDecoder;
  private readonly host: string;

  constructor(ecd: EncoderDecoder, host: string) {
    this.ecd = ecd;
    this.host = host;
  }

  async Post<Request, Response>({
    endpoint,
    req,
  }: {
    endpoint: string;
    req: Request;
  }): Promise<Response> {
    const stringData = this.ecd.encode<Request>(req);
    const res = await fetch(this.formatURL({ endpoint }), {
      method: Method.POST,
      headers: this.getHeaders(),
      body: stringData,
    });
    await this.checkResponse(res);
    return this.ecd.decode<Response>(await res.text());
  }

  async Get<Response>({
    endpoint,
    params,
  }: {
    endpoint: string;
    params?: Params;
  }): Promise<Response> {
    const url = this.formatURL({ endpoint, params });
    const res = await fetch(url, {
      method: Method.GET,
      headers: this.getHeaders(),
    });
    await this.checkResponse(res);
    return this.ecd.decode<Response>(await res.text());
  }

  getHeaders(): HeadersInit {
    return {
      ...this.ecd.headers,
      ...this.token.headers,
    };
  }

  formatURL({
    endpoint,
    params,
  }: {
    endpoint: string;
    params?: Params;
  }): string {
    let url = `${this.host}${endpoint}`;
    if (params) {
      url += "?";
      for (const key in params) {
        url += `${key}=${params[key]}&`;
      }
      url = url.slice(0, -1);
    }
    return url;
  }

  async checkResponse(res: Response): Response {
    if (res.status >= 400) {
      throw {
        error: new Error(`${res.status}`),
        status: res.status,
        ...this.ecd.decode<ErrorResponse>(await res.text()),
      };
    }
  }
}

export interface EncoderDecoder extends BaseEncoderDecoder, HeaderProvider {}

export interface HeaderProvider {
  headers: HeadersInit;
}

type ErrorResponse = {
  errors: TypedError[];
};
