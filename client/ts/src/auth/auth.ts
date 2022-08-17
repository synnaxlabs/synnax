import { User } from "../user/user";

export type Credentials = {
  username: string;
  password: string;
};

export type Token = string;

export type LoginRequest = Credentials;

export type AuthenticationResponse = {
  user: User;
  token: Token;
};

export class Authentication {
  public user: User;
  private transport: AuthenticationTransport;

  constructor(transport: AuthenticationTransport) {
    this.transport = transport;
  }

  async login(request: LoginRequest): Promise<AuthenticationResponse> {
    const res = await this.transport.login(request);
    this.transport.bindCredentials(res.token);
    this.user = res.user;
    return res;
  }
}

export interface AuthenticationTransport {
  login(request: LoginRequest): Promise<AuthenticationResponse>;
  bindCredentials(token: Token): void;
}
