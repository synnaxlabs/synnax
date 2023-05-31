export interface WorkerSetPropsMessage {
  variant: "setState";
  path: string[];
  type: string;
  state: any;
}

export interface WorkerDeleteMessage {
  variant: "delete";
  path: string[];
}

export interface WorkerBootstrapMessage {
  variant: "bootstrap";
  data: any;
}

export type WorkerMessage =
  | WorkerSetPropsMessage
  | WorkerDeleteMessage
  | WorkerBootstrapMessage;
