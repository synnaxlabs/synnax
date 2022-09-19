import {
  createContext,
  PropsWithChildren,
  ReactComponentElement,
  useCallback,
  useContext,
  useEffect,
  useState,
} from "react";
import { useArrowKeys } from "./hooks";
import { Logo, usePersistedState } from "@synnaxlabs/pluto";
import "./Presentation.css";
import clsx from "clsx";

const PresentationContext = createContext<{
  slide: number;
  transition: number;
}>({
  slide: 0,
  transition: 0,
});

export const usePresentationContext = () => useContext(PresentationContext);

export type SlideProps = PropsWithChildren<{
  title: string;
  transitionCount?: number;
  or?: string;
  logoColor?: "white" | "black" | "gradient" | "auto" | "highContrast" | false;
  logoPosition?: "top" | "bottom";
}>;

export const SlideContainer = ({
  transitionCount = 0,
  logoColor = "auto",
  logoPosition = "top",
  title,
  children,
}: SlideProps) => {
  return (
    <div style={{ width: "100vw", height: "100vh", display: "flex" }}>
      {logoColor && (
        <Logo
          height={72}
          className={clsx(
            "slide-container__logo",
            `slide-container__logo--${logoPosition}`
          )}
          color={logoColor}
        />
      )}
      {children}
    </div>
  );
};

type PresentationProps = {
  children: ReactComponentElement<typeof SlideContainer, SlideProps>[];
};

export default function Presentation({ children }: PresentationProps) {
  const [currentSlideIndex, setCurrentSlideIndex] = usePersistedState(
    "presentation-currentSlideIndex",
    0
  );
  const [currentTransition, setCurrentTransition] = usePersistedState(
    "presentation-currentTransition",
    0
  );
  const numSlides = children.length;
  const currentSlide = children[currentSlideIndex];
  const { transitionCount = 0 } = currentSlide.props;

  useEffect(() => {
    window.document.title = currentSlide.props.title;
  }, [currentSlide]);

  const next = useCallback(() => {
    if (currentTransition >= transitionCount - 1) {
      if (currentSlideIndex >= numSlides - 1) return;
      console.log("incrementing slide", currentSlideIndex, numSlides);
      setCurrentSlideIndex(currentSlideIndex + 1);
      setCurrentTransition(0);
    } else {
      setCurrentTransition(currentTransition + 1);
    }
  }, [currentSlideIndex, currentTransition, transitionCount, numSlides]);

  const previous = useCallback(() => {
    if (currentTransition > 0) {
      setCurrentTransition(currentTransition - 1);
    } else {
      if (currentSlideIndex <= 0) return;
      setCurrentSlideIndex(currentSlideIndex - 1);
      setCurrentTransition(
        (children[currentSlideIndex - 1].props.transitionCount || 1) - 1
      );
    }
  }, [currentSlideIndex, currentTransition, children]);

  useArrowKeys({ right: next, left: previous });

  return (
    <PresentationContext.Provider
      value={{ slide: currentSlideIndex, transition: currentTransition }}
    >
      {currentSlide}
    </PresentationContext.Provider>
  );
}
