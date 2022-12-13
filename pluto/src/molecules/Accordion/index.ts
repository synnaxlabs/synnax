import { Accordion as CoreAccordion } from "./Accordion";
export type { AccordionEntry } from "./Accordion";

type CoreAccordionType = typeof CoreAccordion;

type AccordionType = CoreAccordionType;

export const Accordion = CoreAccordion as AccordionType;
