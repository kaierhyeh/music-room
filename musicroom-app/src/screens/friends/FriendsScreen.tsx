import React, { useEffect, useState, useCallback } from 'react';
import { FlatList, View, Pressable, StyleSheet, RefreshControl } from 'react-native';
import { useNavigation } from '@react-navigation/native';
import { friendsApi } from '../../api/friends';
import { apiError } from '../../api/error';
import { User } from '../../api/users';
import Screen from '../../components/ui/Screen';
import Text from '../../components/ui/Text';
import Button from '../../components/ui/Button';
import { colors, spacing } from '../../theme';

export default function FriendsScreen() {
  const nav = useNavigation<any>();
  const [friends, setFriends] = useState<User[]>([]);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    setRefreshing(true);
    try {
      setFriends(await friendsApi.list());
      setError('');
    } catch (e) {
      setError(apiError(e, 'COULD NOT LOAD THE FRIEND LIST'));
    }
    setRefreshing(false);
  }, []);

  useEffect(() => { load(); }, []);

  return (
    <Screen title="FRIENDS" padded={false}>
      <View style={styles.actions}>
        <Button label="REQUESTS" onPress={() => nav.navigate('FriendRequests')} variant="ghost" />
        <Button label="FIND USER" onPress={() => nav.navigate('UserSearch')} variant="ghost" />
      </View>
      {error && friends.length > 0 ? (
        <Text danger style={styles.staleError}>{`! ${error}`}</Text>
      ) : null}
      <FlatList
        data={friends}
        keyExtractor={(i) => i.id}
        refreshControl={<RefreshControl refreshing={refreshing} onRefresh={load} tintColor={colors.accent} />}
        contentContainerStyle={styles.list}
        ListEmptyComponent={
          error
            ? <Text danger style={styles.empty}>{`! ${error}`}</Text>
            : <Text dim style={styles.empty}>{'// NO FRIENDS YET. USE FIND USER TO ADD SOMEONE.'}</Text>
        }
        renderItem={({ item }) => (
          <Pressable
            style={styles.item}
            onPress={() => nav.navigate('UserProfile', { userId: item.id })}
          >
            <Text size="sm" accent>{'> '}</Text>
            <Text size="sm">{item.display_name}</Text>
          </Pressable>
        )}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  actions: { flexDirection: 'row', paddingHorizontal: spacing.md, gap: 8 },
  list: { padding: spacing.md },
  item: { flexDirection: 'row', paddingVertical: spacing.sm, borderBottomWidth: 1, borderBottomColor: colors.border },
  empty: { marginTop: spacing.lg },
  staleError: { paddingHorizontal: spacing.md, paddingTop: spacing.xs },
});
