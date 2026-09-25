import React from 'react';
import { View, Pressable, StyleSheet } from 'react-native';
import { useNavigation } from '@react-navigation/native';
import { useAuth } from '../context/AuthContext';
import Text from './ui/Text';

export default function PlaylistByline({
  ownerId,
  ownerName,
}: {
  ownerId: string;
  ownerName: string;
}) {
  const nav = useNavigation<any>();
  const { user } = useAuth();

  if (user?.id === ownerId) {
    return (
      <View style={styles.row}>
        <Text size="xs" dim>{'// BY '}</Text>
        <Text size="xs" accent>{'ME'}</Text>
      </View>
    );
  }

  if (!ownerName) return null;

  return (
    <Pressable
      onPress={() => nav.navigate('UserProfile', { userId: ownerId })}
      style={styles.row}
      accessibilityRole="link"
      accessibilityLabel={`Open ${ownerName}'s profile`}
    >
      <Text size="xs" dim>{'// BY '}</Text>
      <Text size="xs" accent>{ownerName}</Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: 'row', alignItems: 'center', alignSelf: 'flex-start', paddingVertical: 2 },
});
