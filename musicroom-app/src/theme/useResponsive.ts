import { useWindowDimensions } from 'react-native';

export const WIDE_BREAKPOINT = 900;

export const CONTENT_MAX_WIDTH = 820;

export function useResponsive() {
  const { width, height } = useWindowDimensions();
  return { width, height, wide: width >= WIDE_BREAKPOINT };
}
