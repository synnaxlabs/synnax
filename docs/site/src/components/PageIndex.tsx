interface Page {
  title: string;
  url: string;
  children?: Page[];
}

const pages: Page[] = [];

export const Index = (): JSX.Element => {};
