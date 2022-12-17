import { useRef, useState } from "react";

import { CoreTextProps, Text } from "./Text";

export interface TextEditableProps extends CoreTextProps {
  /* The text to display */
  text: string;
  /* The handler to call when the text changes */
  onChange?: (newText: string) => void;
}

export const TextEditable = ({
  text,
  onChange,
  ...props
}: TextEditableProps): JSX.Element => {
  const [editable, setEditable] = useState(false);
  const ref = useRef<HTMLParagraphElement>(null);

  return (
    <Text
      ref={ref}
      style={{ userSelect: "none", cursor: "pointer" }}
      onBlur={() => setEditable(false)}
      onKeyDown={(e) => ["Enter", "Escape"].includes(e.key) && setEditable(false)}
      onKeyUp={(e) => editable && onChange?.(e.currentTarget.innerText)}
      onDoubleClick={() => setEditable(true)}
      contentEditable={editable}
      {...props}
    >
      {text}
    </Text>
  );
};
