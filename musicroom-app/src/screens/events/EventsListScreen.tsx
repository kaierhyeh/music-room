import React, { useState, useCallback } from 'react';
import { FlatList, View, Pressable, StyleSheet, RefreshControl } from 'react-native';
import { useNavigation, useFocusEffect } from '@react-navigation/native';
import { eventsApi, MusicEvent } from '../../api/events';
import { apiError } from '../../api/error';
import { useAuth } from '../../context/AuthContext';
import { formatWindow, voteWindow } from './eventFormat';
import Screen from '../../components/ui/Screen';
import Text from '../../components/ui/Text';
import Button from '../../components/ui/Button';
import { colors, spacing } from '../../theme';

const LICENSE_TAG: Record<string, string> = {
  open: 'OPEN',
  invited_only: 'INVITE',
  time_and_location: 'T+L',
};

export default function EventsListScreen() {
  const nav = useNavigation<any>();
  const [events, setEvents] = useState<MusicEvent[]>([]);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState('');
  const { user } = useAuth();

  const load = useCallback(async () => {
    setRefreshing(true);
    try {
      setEvents(await eventsApi.list());
      setError('');
    } catch (e) {
      setError(apiError(e, 'COULD NOT LOAD THE EVENTS'));
    }
    setRefreshing(false);
  }, []);

  useFocusEffect(useCallback(() => { load(); }, [load]));

  return (
    <Screen title="EVENTS" padded={false}>
      <View style={styles.fab}>
        <Button label="+ NEW EVENT" onPress={() => nav.navigate('CreateEvent')} />
      </View>
      {error && events.length > 0 ? (
        <Text danger style={styles.staleError}>{`! ${error}`}</Text>
      ) : null}
      <FlatList
        data={events}
        keyExtractor={(e) => e.id}
        refreshControl={<RefreshControl refreshing={refreshing} onRefresh={load} tintColor={colors.accent} />}
        contentContainerStyle={styles.list}
        ListEmptyComponent={
          error
            ? <Text danger style={styles.empty}>{`! ${error}`}</Text>
            : <Text dim style={styles.empty}>{'// NO EVENTS YET. CREATE ONE TO GET STARTED.'}</Text>
        }
        renderItem={({ item }) => (
          <Pressable style={styles.item} onPress={() => nav.navigate('EventDetail', { eventId: item.id })}>
            <View style={styles.itemHeader}>
              <Text size="sm" accent>{`> ${item.name}`}</Text>
              <Text size="xs" dim>{LICENSE_TAG[item.license]}</Text>
            </View>
            {item.description && <Text size="xs" dim numberOfLines={1}>{item.description}</Text>}
            {item.vote_start_at && item.vote_end_at ? (
              <Text size="xs" dim>
                {`// ${formatWindow(item.vote_start_at, item.vote_end_at)}${voteWindow(item) === 'open' ? ' — LIVE NOW' : voteWindow(item) === 'closed' ? ' — OVER' : ''}`}
              </Text>
            ) : null}
            <View style={styles.itemHeader}>
              {item.owner_id === user?.id ? (
                <Text size="xs" accent>{'// HOSTED BY ME'}</Text>
              ) : (
                <Text size="xs" dim>{`// HOSTED BY ${item.owner_display_name ?? 'SOMEONE'}`}</Text>
              )}
              {item.visibility === 'private' ? <Text size="xs" dim>{'PRIVATE'}</Text> : null}
            </View>
          </Pressable>
        )}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  fab: { paddingHorizontal: spacing.md },
  list: { padding: spacing.md },
  item: { paddingVertical: spacing.sm, borderBottomWidth: 1, borderBottomColor: colors.border },
  itemHeader: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center' },
  empty: { marginTop: spacing.lg },
  staleError: { paddingHorizontal: spacing.md, paddingTop: spacing.xs },
});
