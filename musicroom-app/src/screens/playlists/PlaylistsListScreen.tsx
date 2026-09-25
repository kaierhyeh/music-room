import React, { useState, useCallback } from 'react';
import { FlatList, View, Pressable, StyleSheet, RefreshControl } from 'react-native';
import { useNavigation, useFocusEffect, useRoute, RouteProp } from '@react-navigation/native';
import type { PlaylistsStackParams } from '../../navigation/AppNavigator';
import { playlistsApi, Playlist } from '../../api/playlists';
import { apiError } from '../../api/error';
import Screen from '../../components/ui/Screen';
import Text from '../../components/ui/Text';
import Button from '../../components/ui/Button';
import PlaylistByline from '../../components/PlaylistByline';
import { colors, spacing } from '../../theme';

type Tab = 'all' | 'mine' | 'collabs';

const TABS: { key: Tab; label: string }[] = [
  { key: 'all', label: 'ALL' },
  { key: 'mine', label: 'MY PLAYLISTS' },
  { key: 'collabs', label: 'MY COLLABS' },
];

const EMPTY: Record<Tab, string> = {
  all: '// NO PLAYLISTS YET. CREATE THE FIRST ONE.',
  mine: "// YOU DON'T HAVE A PLAYLIST YET. CREATE ONE TO GET STARTED.",
  collabs: '// NO SHARED PLAYLISTS YET. WHEN A FRIEND INVITES YOU, IT SHOWS UP HERE.',
};

const FETCH: Record<Tab, () => Promise<Playlist[]>> = {
  all: playlistsApi.list,
  mine: playlistsApi.listMine,
  collabs: playlistsApi.listShared,
};

function isTab(v: unknown): v is Tab {
  return TABS.some((t) => t.key === v);
}

export default function PlaylistsListScreen() {
  const nav = useNavigation<any>();
  const { params } = useRoute<RouteProp<PlaylistsStackParams, 'PlaylistsList'>>();
  const tab: Tab = isTab(params?.tab) ? params.tab : 'all';
  const [playlists, setPlaylists] = useState<Playlist[]>([]);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    setRefreshing(true);
    try {
      setPlaylists(await FETCH[tab]());
      setError('');
    } catch (e) {
      setError(apiError(e, 'COULD NOT LOAD THE PLAYLISTS'));
    }
    setRefreshing(false);
  }, [tab]);

  useFocusEffect(useCallback(() => { load(); }, [load]));

  function switchTab(next: Tab) {
    if (next === tab) return;
    setPlaylists([]);
    setError('');
    nav.setParams({ tab: next === 'all' ? undefined : next });
  }

  return (
    <Screen title="PLAYLISTS" padded={false}>
      <View style={styles.tabs}>
        {TABS.map((t) => (
          <Pressable key={t.key} onPress={() => switchTab(t.key)} style={[styles.tab, tab === t.key && styles.tabActive]}>
            <Text size="sm" accent={tab === t.key} dim={tab !== t.key}>{t.label}</Text>
          </Pressable>
        ))}
      </View>

      <View style={styles.fab}>
        <Button label="+ NEW PLAYLIST" onPress={() => nav.navigate('CreatePlaylist')} />
      </View>
      {error && playlists.length > 0 ? (
        <Text danger style={styles.staleError}>{`! ${error}`}</Text>
      ) : null}
      <FlatList
        data={playlists}
        keyExtractor={(p) => p.id}
        refreshControl={<RefreshControl refreshing={refreshing} onRefresh={load} tintColor={colors.accent} />}
        contentContainerStyle={styles.list}
        ListEmptyComponent={
          error
            ? <Text danger style={styles.empty}>{`! ${error}`}</Text>
            : <Text dim style={styles.empty}>{EMPTY[tab]}</Text>
        }
        renderItem={({ item }) => (
          <Pressable style={styles.item} onPress={() => nav.navigate('PlaylistDetail', { playlistId: item.id })}>
            <View style={styles.row}>
              <Text size="sm" accent>{`> ${item.name}`}</Text>
              <Text size="xs" dim>{`v${item.version}`}</Text>
            </View>
            {item.description && <Text size="xs" dim numberOfLines={1}>{item.description}</Text>}
            {tab === 'mine' ? (
              <View style={styles.row}>
                <View style={styles.metaLine}>
                  <PlaylistByline ownerId={item.owner_id} ownerName={item.owner_display_name} />
                  <Text size="xs" dim>
                    {` — ${item.visibility.toUpperCase()} — ${item.license.replace('_', ' ').toUpperCase()}`}
                  </Text>
                </View>
                <Pressable
                  onPress={() => nav.navigate('EditPlaylist', { playlistId: item.id })}
                  style={styles.editBtn}
                  accessibilityRole="button"
                  accessibilityLabel={`Edit ${item.name}`}
                >
                  <Text size="xs" accent>{'[ EDIT ]'}</Text>
                </Pressable>
              </View>
            ) : tab === 'collabs' ? (
              <View style={styles.row}>
                <View style={styles.metaLine}>
                  <PlaylistByline ownerId={item.owner_id} ownerName={item.owner_display_name} />
                  <Text size="xs" dim>{` — ${item.visibility.toUpperCase()}`}</Text>
                </View>
                <Text size="xs" accent={item.can_edit} dim={!item.can_edit}>
                  {item.can_edit ? '[EDIT]' : '[VIEW]'}
                </Text>
              </View>
            ) : (
              <PlaylistByline ownerId={item.owner_id} ownerName={item.owner_display_name} />
            )}
          </Pressable>
        )}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  tabs: { flexDirection: 'row', borderBottomWidth: 1, borderBottomColor: colors.border },
  tab: { flex: 1, alignItems: 'center', paddingVertical: spacing.sm, paddingHorizontal: spacing.xs, borderBottomWidth: 2, borderBottomColor: 'transparent' },
  tabActive: { borderBottomColor: colors.accent, backgroundColor: colors.accentGlow },
  fab: { paddingHorizontal: spacing.md, paddingTop: spacing.sm },
  list: { padding: spacing.md },
  item: { paddingVertical: spacing.sm, borderBottomWidth: 1, borderBottomColor: colors.border },
  row: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center' },
  metaLine: { flexDirection: 'row', alignItems: 'center', flexShrink: 1 },
  editBtn: { paddingVertical: 2, paddingLeft: spacing.sm },
  empty: { marginTop: spacing.lg },
  staleError: { paddingHorizontal: spacing.md, paddingTop: spacing.xs },
});
