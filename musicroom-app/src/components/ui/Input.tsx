import React, { useState } from 'react';
import { TextInput, TextInputProps, View, StyleSheet } from 'react-native';
import { colors, fonts, fontSize, spacing } from '../../theme';
import Text from './Text';

interface Props extends TextInputProps {
  label?: string;
  error?: string;
}

export default function Input({ label, error, style, ...rest }: Props) {
  const [focused, setFocused] = useState(false);

  return (
    <View style={styles.wrapper}>
      {label && (
        <Text size="sm" dim style={styles.label}>
          {`// ${label.toUpperCase()}`}
        </Text>
      )}
      <TextInput
        style={[
          styles.input,
          focused && styles.focused,
          error ? styles.errorBorder : undefined,
          style,
        ]}
        placeholderTextColor={colors.textDim}
        onFocus={() => setFocused(true)}
        onBlur={() => setFocused(false)}
        {...rest}
      />
      {error && (
        <Text size="sm" danger>
          {`! ${error}`}
        </Text>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  wrapper: {
    marginVertical: spacing.xs,
  },
  label: {
    marginBottom: 2,
  },
  input: {
    borderWidth: 1,
    borderColor: colors.border,
    backgroundColor: colors.surface,
    color: colors.text,
    fontFamily: fonts.mono,
    fontSize: fontSize.md,
    paddingVertical: spacing.sm,
    paddingHorizontal: spacing.md,
    letterSpacing: 1,
  },
  focused: {
    borderColor: colors.accent,
    backgroundColor: colors.accentGlow,
  },
  errorBorder: {
    borderColor: colors.danger,
  },
});
