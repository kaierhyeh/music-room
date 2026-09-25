import React from 'react';
import { Text as RNText, TextProps, StyleSheet } from 'react-native';
import { colors, fonts, fontSize } from '../../theme';

interface Props extends TextProps {
  size?: keyof typeof fontSize;
  dim?: boolean;
  danger?: boolean;
  accent?: boolean;
  upper?: boolean;
}

export default function Text({
  size = 'md',
  dim,
  danger,
  accent,
  upper,
  style,
  children,
  ...rest
}: Props) {
  const color = danger
    ? colors.danger
    : accent
    ? colors.accent
    : dim
    ? colors.textDim
    : colors.text;

  return (
    <RNText
      style={[
        styles.base,
        { fontSize: fontSize[size], color },
        style,
      ]}
      {...rest}
    >
      {upper && typeof children === 'string' ? children.toUpperCase() : children}
    </RNText>
  );
}

const styles = StyleSheet.create({
  base: {
    fontFamily: fonts.mono,
    letterSpacing: 1,
  },
});
