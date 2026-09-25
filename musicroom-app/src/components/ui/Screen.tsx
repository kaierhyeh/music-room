import React from 'react';
import { ScrollView, View, StyleSheet, StatusBar, KeyboardAvoidingView, Platform } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { colors, spacing } from '../../theme';
import { CONTENT_MAX_WIDTH, useResponsive } from '../../theme/useResponsive';
import Scanlines from '../Scanlines';
import Text from './Text';

interface Props {
  title?: string;
  children: React.ReactNode;
  scroll?: boolean;
  padded?: boolean;
}

export default function Screen({ title, children, scroll = false, padded = true }: Props) {
  const { wide } = useResponsive();

  const inner = (
    <View style={[styles.inner, padded && styles.padded, wide && styles.column]}>
      {title && (
        <View style={styles.header}>
          <Text size="lg" accent upper style={styles.title}>
            {`■ ${title}`}
          </Text>
          <View style={styles.divider} />
        </View>
      )}
      {children}
    </View>
  );

  return (
    <SafeAreaView style={styles.root}>
      <StatusBar barStyle="light-content" backgroundColor={colors.bg} />
      <Scanlines />
      <KeyboardAvoidingView
        style={styles.root}
        behavior="padding"
        enabled={Platform.OS !== 'web'}
      >
        {scroll ? (
          <ScrollView
            style={styles.scroll}
            contentContainerStyle={!padded ? undefined : undefined}
            showsVerticalScrollIndicator={false}
            keyboardShouldPersistTaps="handled"
          >
            {inner}
          </ScrollView>
        ) : (
          inner
        )}
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  root: {
    flex: 1,
    backgroundColor: colors.bg,
  },
  scroll: {
    flex: 1,
  },
  inner: {
    flex: 1,
  },
  padded: {
    padding: spacing.md,
  },
  column: {
    width: '100%',
    maxWidth: CONTENT_MAX_WIDTH,
    alignSelf: 'center',
  },
  header: {
    marginBottom: spacing.md,
  },
  title: {
    marginBottom: spacing.xs,
  },
  divider: {
    height: 1,
    backgroundColor: colors.accentDim,
  },
});
