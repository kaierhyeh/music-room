import React from 'react';
import { View, StyleSheet } from 'react-native';
import { MusicPreferences } from '../api/users';
import Text from './ui/Text';
import { spacing } from '../theme';

export default function MusicPreferencesBlock({ prefs }: { prefs?: MusicPreferences }) {
  const genres = prefs?.genres ?? [];
  const artists = prefs?.artists ?? [];

  return (
    <View style={styles.field}>
      <Text size="sm" dim>{'// MUSIC PREFERENCES'}</Text>
      {genres.length === 0 && artists.length === 0 ? (
        <Text size="sm" dim>{'// NOTHING SET'}</Text>
      ) : (
        <>
          {genres.length > 0 && <Text size="sm" accent>{genres.join(' / ').toUpperCase()}</Text>}
          {artists.length > 0 && <Text size="sm">{artists.join(', ')}</Text>}
        </>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  field: { marginVertical: spacing.xs },
});
