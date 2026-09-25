import React, { useEffect, useState } from 'react';
import { FlatList, View, StyleSheet } from 'react-native';
import { friendsApi, FriendRequest } from '../../api/friends';
import { apiError } from '../../api/error';
import Screen from '../../components/ui/Screen';
import Text from '../../components/ui/Text';
import Button from '../../components/ui/Button';
import { colors, spacing } from '../../theme';

export default function FriendRequestsScreen() {
  const [requests, setRequests] = useState<FriendRequest[]>([]);
  const [error, setError] = useState('');

  async function load() {
    try {
      setRequests(await friendsApi.requests());
      setError('');
    } catch (e) {
      setError(apiError(e, 'COULD NOT LOAD THE REQUESTS'));
    }
  }

  useEffect(() => { load(); }, []);

  async function accept(id: string) {
    if (!(await answer(() => friendsApi.accept(id), 'COULD NOT ACCEPT'))) return;
    load();
  }

  async function reject(id: string) {
    if (!(await answer(() => friendsApi.reject(id), 'COULD NOT REJECT'))) return;
    load();
  }

  async function answer(call: () => Promise<unknown>, fallback: string): Promise<boolean> {
    setError('');
    try {
      await call();
      return true;
    } catch (e) {
      setError(apiError(e, fallback));
      return false;
    }
  }

  return (
    <Screen title="FRIEND REQUESTS">
      {error && requests.length > 0 ? <Text danger>{`! ${error}`}</Text> : null}
      <FlatList
        data={requests}
        keyExtractor={(i) => i.id}
        ListEmptyComponent={
          error ? (
            <View>
              <Text danger>{`! ${error}`}</Text>
              <View style={{ height: spacing.md }} />
              <Button label="RETRY" onPress={load} />
            </View>
          ) : (
            <Text dim>{'// NO PENDING REQUESTS'}</Text>
          )
        }
        renderItem={({ item }) => (
          <View style={styles.item}>
            <Text size="sm" accent>{`> ${item.display_name}`}</Text>
            <View style={styles.btns}>
              <Button label="ACCEPT" onPress={() => accept(item.id)} />
              <Button label="REJECT" onPress={() => reject(item.id)} variant="danger" />
            </View>
          </View>
        )}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  item: { paddingVertical: spacing.sm, borderBottomWidth: 1, borderBottomColor: colors.border },
  btns: { flexDirection: 'row', gap: 8, marginTop: spacing.xs },
});
