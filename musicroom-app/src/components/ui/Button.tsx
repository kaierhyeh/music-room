import React from 'react';
import { Pressable, StyleSheet, View } from 'react-native';
import { colors, fonts, fontSize, spacing } from '../../theme';
import Text from './Text';

interface Props {
  label: string;
  onPress: () => void;
  variant?: 'primary' | 'danger' | 'ghost';
  disabled?: boolean;
}

export default function Button({ label, onPress, variant = 'primary', disabled }: Props) {
  const borderColor =
    variant === 'danger' ? colors.danger : variant === 'ghost' ? colors.border : colors.accent;
  const textColor =
    variant === 'danger' ? colors.danger : variant === 'ghost' ? colors.textDim : colors.accent;

  return (
    <Pressable
      onPress={onPress}
      disabled={disabled}
      style={({ pressed }) => [
        styles.btn,
        { borderColor },
        pressed && styles.pressed,
        disabled && styles.disabled,
      ]}
    >
      {({ pressed }) => (
        <Text size="md" style={{ color: disabled ? colors.textDisabled : textColor }}>
          {pressed ? `> ${label.toUpperCase()}` : `  ${label.toUpperCase()}`}
        </Text>
      )}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  btn: {
    borderWidth: 1,
    paddingVertical: spacing.sm,
    paddingHorizontal: spacing.md,
    marginVertical: spacing.xs,
  },
  pressed: {
    backgroundColor: colors.accentGlow,
  },
  disabled: {
    borderColor: colors.border,
    opacity: 0.4,
  },
});
